# Pollify — справочник по проекту

Сервис опросов и голосований с поддержкой анонимных опросов, множественного выбора, свободных текстовых ответов и кворумной модерации сообществом. Бэкенд на Go, фронтенд на React/TypeScript, хранение в PostgreSQL 17, упаковано в Docker-контейнеры.

Этот документ — развёрнутый справочник. Краткий обзор проекта см. в `.kilo/plans/TLDR.md`. Инструкции по развёртыванию — в `DEPLOY.md`. Исходный план реализации — в `.kilo/plans/1776941384248-brave-moon.md`.

---

## 1. Цели и принципы

Сервис существует, чтобы собирать мнения и проводить голосования. План фиксирует небольшой набор принципов, который определяет каждое архитектурное решение:

| Принцип | Что это значит на практике |
| --- | --- |
| **Разделение участия и выбора** | Факт «пользователь X участвовал в опросе Y» хранится в одной таблице; сами голоса — в другой. Это позволяет выразить анонимность на уровне схемы, а не только API. |
| **Универсальная модель голоса** | Одна таблица `votes` представляет и выбор варианта, и свободный текст — приложение выбирает, какую колонку заполнить. Множественный выбор — это просто несколько строк в `votes` с одинаковым `(poll_id, user_id)`. |
| **Конфигурируемые опросы** | Анонимность, множественный выбор (с `max_choices`) и свободный текст — флаги на `polls`. Множественный выбор и свободный текст в v1 **взаимоисключают** друг друга, чтобы агрегация результатов оставалась простой. |
| **Ограниченная модерация** | Админы не могут редактировать контент напрямую. Они действуют только по жалобам пользователей и только в составе кворума. |
| **Кворумная модерация** | Один админ не может закрыть жалобу. Два совпадающих решения (по умолчанию) закрывают её; второй админ видит первое решение, но не может рассматривать собственную жалобу. |
| **Гарантии доверия, не зависящие от админ-доступа** | Анонимные голоса нельзя связать с пользователями через API. Поданные голоса неизменяемы. |

---

## 2. Архитектура

```
┌──────────────────┐  HTTP / JSON   ┌──────────────────┐  pgx/v5    ┌──────────────────┐
│ Frontend SPA     │  Bearer JWT    │ Backend API      │  пул       │ PostgreSQL 17    │
│ React + TS       │ ─────────────► │ Go-монолит       │ ─────────► │ 7 таблиц         │
│ react-router-dom │                │ chi + pgx + jwt  │            │ 3 триггера       │
│ Vite (dev)       │                │ гексагональная   │            │ 9 индексов       │
└──────────────────┘                └──────────────────┘            └──────────────────┘
        │                                   │
        ▼                                   ▼
   в проде раздаёт Caddy             backend/migrations/
   (proxy /api/*)                    одна .up.sql / одна .down.sql
```

### Структура бэкенда

Бэкенд — один Go-бинарник, разделённый по **бизнес-компонентам**, а не по техническим слоям. Каждый компонент лежит непосредственно в `internal/`:

```
backend/
├── cmd/api/main.go               – точка входа; вызывает app.Run
├── internal/
│   ├── users/        core/{models,ports,errors,service}
│   │                 adapters/{postgres, http}
│   ├── polls/        core/{models,ports,errors,service,validators}
│   │                 adapters/{postgres, http}
│   ├── voting/       core/{models,ports,errors,service,policies}
│   │                 adapters/{postgres, http}
│   ├── moderation/   core/{models,ports,errors,service,quorum}
│   │                 adapters/{postgres, http}
│   ├── platform/     auth/         (JWT-issuer, HMAC-verifier, role guard)
│   │                 httpserver/   (обёртка над жизненным циклом HTTP)
│   │                 postgres/     (pgx-пул + миграции)
│   │                 app/          (сборка роутера)
│   │                 config/, clock/, logging/, openapi/, postgrestest/
│   └── pkg/apierror/             – единый ErrorResponse
├── migrations/                    – нумерованные SQL-миграции
└── tests/integration/api_test.go – e2e-смоук против реальной БД
```

Каждый пакет `core/` содержит чистые доменные типы: модели, порты (интерфейсы), сервисы (use cases), ошибки, валидаторы и политики. Адаптеры зависят от интерфейсов из `core/` — направление инвертировано. Слой platform всё это собирает.

### Структура фронтенда

```
frontend/
├── src/
│   ├── api/              – TypeScript-типы по OpenAPI + fetch-обёртка
│   ├── auth/             – AuthContext (токен в localStorage, гидратация /me)
│   ├── pages/            – Login, Register, PollList, CreatePoll,
│   │                       PollDetail, PollResults, ModerationQueue, ReportDetail
│   ├── App.tsx           – маршруты + RequireAuth / RequireAdmin
│   ├── main.tsx          – BrowserRouter
│   └── index.css         – дизайн-система (CSS custom properties, плашки и т.д.)
├── tests/e2e/            – Playwright-спеки (auth, poll-flow, moderation)
├── playwright.config.ts
├── Dockerfile            – multi-stage: node-сборка → caddy-раздача
├── Caddyfile             – reverse-proxy /api + SPA fallback
└── vite.config.ts        – dev-proxy /api → 127.0.0.1:8080
```

---

## 3. Доменная модель и схема БД

### 3.1 Таблицы

| Таблица | Назначение | Ключевые столбцы / ограничения |
| --- | --- | --- |
| `users` | Идентичность + роль | `email UNIQUE`, `role IN ('USER','ADMIN')` |
| `polls` | Метаданные опроса | Расписание, флаги анонимности, multi-choice, custom-answer. Три CHECK (см. ниже). |
| `options` | Варианты для опросов с фиксированным выбором | `poll_id` FK с `ON DELETE CASCADE` |
| `poll_participants` | Факт участия пользователя | **PK = (poll_id, user_id)** — инвариант «один пользователь, один голос» |
| `votes` | Сам голос | `option_id` **XOR** `custom_text`; `user_id` может быть NULL |
| `reports` | Жалобы пользователей | `status` — Postgres-enum |
| `report_reviews` | Решения админов по жалобам | **PK = (report_id, admin_id)** — один review на админа на жалобу |

### 3.2 ENUM-типы

```sql
CREATE TYPE report_status   AS ENUM ('OPEN', 'IN_REVIEW', 'RESOLVED', 'REJECTED');
CREATE TYPE review_decision AS ENUM ('APPROVE', 'REJECT');
```

Это настоящие Postgres-enum-типы — доменная типизация на уровне хранения.

### 3.3 CHECK-ограничения

```sql
-- polls
CHECK (end_at > start_at)
CHECK (NOT is_multiple_choice OR max_choices > 1)
CHECK (NOT (is_multiple_choice AND allow_custom_answer))

-- votes (взаимоисключение типов payload)
CHECK (
    (option_id IS NOT NULL AND custom_text IS NULL) OR
    (option_id IS NULL     AND custom_text IS NOT NULL)
)

-- users
CHECK (role IN ('USER', 'ADMIN'))
```

### 3.4 Триггеры

Три PL/pgSQL-триггера выступают второй линией обороны — срабатывают даже при прямом доступе к БД:

| Триггер | Что обеспечивает |
| --- | --- |
| `report_reviews_admin_only` | На каждом INSERT/UPDATE проверяет, что роль актора = `ADMIN`. |
| `report_reviews_no_self_review` | Запрещает review, где `admin_id = reports.created_by` для этой жалобы. |
| `votes_option_matches_poll` | Не-NULL `votes.option_id` должен ссылаться на строку `options` с тем же `poll_id`. |

### 3.5 Индексы

```sql
polls(created_by)
polls(is_hidden, start_at, end_at)        -- фильтры списка опросов
options(poll_id)
votes(poll_id), votes(user_id)
reports(poll_id), reports(created_by), reports(status)
```

### 3.6 Анонимность через структуру

Это главное архитектурное решение, заслуживающее подробного разбора.

```
poll_participants                  votes
─────────────────                  ─────
poll_id, user_id, voted_at         id, poll_id, option_id|custom_text, user_id?, created_at
PK (poll_id, user_id)              user_id NULLABLE
```

Для **неанонимных опросов** приложение записывает `votes.user_id = актор`. Для **анонимных опросов** оставляет `votes.user_id = NULL`. В обоих случаях `poll_participants` фиксирует факт участия, поэтому:

- Инвариант «один пользователь — один голос» обеспечивается составным PK.
- Для анонимных опросов ни одна строка в `votes` не привязана к пользователю. Связи нигде не существует.
- API анонимного опроса никогда не возвращает `voter_details` и не принимает `include_voters=true`. Сервис голосования отклоняет такие запросы на доменной границе.

Это разделяет уникальность (PK на `poll_participants`) и авторство (`votes.user_id`) — анонимность становится структурным свойством схемы, а не runtime-проверкой.

---

## 4. Аутентификация и авторизация

### 4.1 Токены

- `POST /api/v1/auth/register` и `POST /api/v1/auth/login` выдают **access-токен** (JWT с HMAC-SHA256) и непрозрачный refresh-токен.
- Время жизни access-токена — 1 час. Refresh-токены в v1 — заглушки, эндпоинта обновления нет.
- Токены содержат `sub` (id пользователя), `role`, `iat`, `nbf`, `exp`.
- Фронтенд хранит обе записи в `localStorage` и отправляет `Authorization: Bearer <access_token>` со всеми защищёнными запросами.

### 4.2 Роли

Только две:

- **`USER`** (по умолчанию) — все аккаунты создаются с этой ролью. `POST /auth/register` жёстко прописывает `role = USER`. В API нет пути, повышающего пользователя до ADMIN.
- **`ADMIN`** — назначается вне приложения тем, у кого есть доступ к БД (SQL `UPDATE users SET role='ADMIN' WHERE email='…'`). После повышения пользователь должен выйти и снова войти, чтобы получить свежий JWT с новой ролью.

Решение оставлять повышение до ADMIN строго вне приложения — намеренное. По плану: админы — это операторы, а не самообслуживаемая категория. Никто не может поднять себя или другого до ADMIN через API.

### 4.3 Middleware

- `auth.Middleware(verifier)` — валидирует bearer-токен и инжектит `users.AuthenticatedUser` в request context.
- `auth.RequireRole(role)` — middleware, возвращающий 403, если роль актора не совпадает. Применяется к маршрутам `/reports/*` и к листингу жалоб по опросу для админов.

---

## 5. Поверхность API

Все пути под `/api/v1`. Полная схема — в `api/openapi.yaml`. Публичные эндпоинты явно отказываются от bearer.

### 5.1 Auth и identity

| Метод | Путь | Доступ | Поведение |
| --- | --- | --- | --- |
| `POST` | `/auth/register` | публично | Email + password (≥8 символов) → 201 с user + tokens. 409, если email занят. |
| `POST` | `/auth/login` | публично | Email + password → 200 с user + tokens. 401 при неверных кредах. |
| `GET`  | `/users/me` | USER | Текущий пользователь как `UserSummary`. |

### 5.2 Опросы

| Метод | Путь | Доступ | Поведение |
| --- | --- | --- | --- |
| `GET` | `/polls/` | USER | Постраничный список с фильтрами (`status`, `creator_id`, `is_anonymous`, `is_multiple_choice`, `allow_custom_answer`, `available_for_voting`, `page`, `limit`, `sort`). |
| `POST` | `/polls/` | USER | Создаёт опрос. Доменный слой отвергает multi-choice + custom-answer, end-before-start и т.п. |
| `GET` | `/polls/{id}` | USER | Один опрос с options и вычисленным статусом. |
| `PATCH` | `/polls/{id}` | USER (создатель) | Title / description / start_at / end_at — только до `start_at` и до появления участников. Иначе 409. |

### 5.3 Голосование

| Метод | Путь | Доступ | Поведение |
| --- | --- | --- | --- |
| `POST` | `/polls/{id}/votes` | USER | Один голос на (poll, user). Тело — `oneOf` `{option_ids: [...]}` или `{custom_text: "..."}`. 409, если уже проголосовал или опрос неактивен. |
| `GET` | `/polls/{id}/results` | USER | Агрегированные счётчики. `?include_voters=true` работает только для неанонимных опросов (иначе 422). |

### 5.4 Жалобы и модерация

| Метод | Путь | Доступ | Поведение |
| --- | --- | --- | --- |
| `POST` | `/polls/{id}/reports` | USER | Одна активная жалоба на (user, poll). 409 при дубле. |
| `GET` | `/polls/{id}/reports` | ADMIN | Жалобы по конкретному опросу. |
| `GET` | `/reports/` | ADMIN | Все жалобы, постранично, с фильтром по `status`. |
| `GET` | `/reports/{id}` | ADMIN | Одна жалоба со всеми review. |
| `POST` | `/reports/{id}/reviews` | ADMIN | Отправляет `APPROVE` / `REJECT`. Запрещено для собственной жалобы (403). Запускает логику кворума. |

### 5.5 Ошибки

Все ошибки используют один формат:

```json
{
  "error": {
    "code": "poll_already_voted",
    "message": "user has already participated in this poll",
    "details": {}
  }
}
```

Коды: `validation_error`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `poll_not_active`, `poll_hidden`, `poll_already_voted`, `poll_vote_payload_invalid`, `poll_update_not_allowed`, `report_already_exists`, `report_review_self_forbidden`, `report_review_already_exists`, `report_already_resolved`, `quorum_not_reached`, `internal_error`.

---

## 6. Кворумная модерация подробно

Логика кворума живёт в `backend/internal/moderation/core/quorum.go` и представляет собой чистую функцию над списком reviews:

```
EvaluateQuorum(reviews, quorum) -> {
    approval_count, rejection_count,
    reached: bool,
    final_decision,
    status: report_status,
    resolution: string
}
```

Состояния и переходы:

```
        OPEN
          │  пришёл первый review
          ▼
        IN_REVIEW
          │
          ├─ approval_count ≥ quorum  ──► RESOLVED   (опрос скрыт)
          └─ rejection_count ≥ quorum ──► REJECTED   (опрос остаётся видимым)
```

По умолчанию кворум — **2** (настраивается в `app.Run` через `moderation.DefaultQuorum`). После достижения кворума жалоба считается закрытой; дальнейшие попытки review возвращают `409 report_already_resolved`.

Решение скрыть опрос (при APPROVE-кворуме) принимается сервисом модерации синхронно — он вызывает `polls.PollVisibilityWriter.Hide(pollID)` и только потом сохраняет review. Всю последовательность ведёт код приложения; БД охраняет то, что может (роль ADMIN, отсутствие self-review, один review на админа), через триггеры.

---

## 7. Стек технологий

| Слой | Выбор | Почему |
| --- | --- | --- |
| Язык бэкенда | Go 1.25 | Статический бинарник, маленький след, мощная стандартная HTTP-библиотека. |
| HTTP-роутинг | `github.com/go-chi/chi/v5` | Идиоматично, легко комбинировать middleware. |
| Драйвер БД | `github.com/jackc/pgx/v5` | Нативный Postgres-протокол, пул, prepared statements. |
| Auth | `github.com/golang-jwt/jwt/v5` | Стандартная JWT-библиотека. |
| Хеш паролей | `golang.org/x/crypto/bcrypt` | Адаптивный cost factor, проверенная реализация. |
| Frontend | React 19 + TypeScript | Индустриальный стандарт; типы помогают на API-границе. |
| Сборка FE | Vite 8 | Быстрый dev-server с нативным ESM, простой proxy. |
| Роутинг (FE) | `react-router-dom` v7 | BrowserRouter для чистых URL. |
| База | PostgreSQL 17 | Сильные ограничения, триггеры, enum-типы. |
| Reverse proxy | Caddy 2 | Один бинарник, авто-HTTPS, простой Caddyfile. |
| Контейнеры | Docker + docker compose v2 | Развёртывание на одном хосте. |
| E2E-тесты | Playwright | Реальный браузер, реальный SPA. |

---

## 8. Стратегия тестирования

Пять слоёв тестов, все зелёные:

| Слой | Что покрывает | Где |
| --- | --- | --- |
| Доменные unit-тесты | Валидаторы и политики (правила payload голоса, правила создания опроса, кворум). | `internal/*/core/*_test.go` (там, где они есть) |
| HTTP-адаптеры unit | Форма запроса/ответа, маппинг ошибок, role guard со stub-сервисами. | `internal/*/adapters/http/handlers_test.go` |
| Postgres-репозитории | Реальный Postgres через хелпер `postgrestest` (отдельная схема на тест). | `internal/*/adapters/postgres/repository_test.go` |
| Backend-интеграция | Полный chi-роутер против реальной БД: register → vote → report → quorum → poll скрыт. | `backend/tests/integration/api_test.go` |
| Frontend E2E | Playwright гоняет Chromium против запущенного prod-стека. 14 спеков покрывают auth, жизненный цикл опроса, multi-choice / анонимные / свободно-текстовые варианты, потоки модерации. | `frontend/tests/e2e/*.spec.ts` |

Запуск всех тестов:

```bash
# backend (postgrestest требует DATABASE_URL)
cd backend && DATABASE_URL='postgres://pollify:pollify@127.0.0.1:5432/pollify?sslmode=disable' \
  go test ./...

# frontend E2E (стек должен быть поднят)
cd frontend && npm run e2e
```

---

## 9. Локальная разработка

### 9.1 Два стека

| Стек | Compose-файл | Сценарий |
| --- | --- | --- |
| Backend dev | `docker-compose.yml` | API + Postgres на хост-портах 8080 / 5432; рядом `npm run dev` в `frontend/` для Vite HMR. |
| Local prod | `docker-compose.prod.yml` | API + Postgres + Caddy, раздающий собранный SPA на порту 80. Ближе всего к настоящему деплою. |

### 9.2 Make-цели

```
make help            # список всех команд
make up              # local-prod-стек + три админа admin1..3@gmail.com / admin123
make down            # стоп, БД сохраняется
make reset           # стереть volume БД + пересоздать админов
make seed-admins     # (пере)создать трёх админов через живой API
make logs            # тейл всех сервисов
make psql            # psql в контейнер БД
make dev-up / dev-down  # только backend dev-стек
```

`make up` создаёт `.env` с дев-дефолтами (если его нет), ждёт `/healthz`, регистрирует трёх админов и сразу же повышает их в БД, чтобы поток модерации можно было показать за секунды.

### 9.3 Готовые админ-аккаунты

| Email | Пароль | Роль |
| --- | --- | --- |
| admin1@gmail.com | admin123 | ADMIN |
| admin2@gmail.com | admin123 | ADMIN |
| admin3@gmail.com | admin123 | ADMIN |

Три — потому что в одной демо-сессии нужно показать: жалобу, поданную обычным пользователем, approve от admin1 (IN_REVIEW), approve от admin2 (RESOLVED + опрос скрыт), и admin3 в запасе, чтобы продемонстрировать, что после resolution дальнейшие review отклоняются.

---

## 10. Развёртывание

Развёртывание на одном хосте задокументировано в `DEPLOY.md`. Кратко:

1. Linux VM с Docker и Docker Compose.
2. `git clone` репозитория; `cp .env.example .env`; заполните `POSTGRES_PASSWORD` (любой надёжный) и `JWT_SECRET` (`openssl rand -hex 32`).
3. `docker compose -f docker-compose.prod.yml up -d --build`.
4. Откройте `http://<server-ip>`.
5. Назначьте первого админа через `docker compose ... exec postgres psql -U pollify -d pollify -c "UPDATE users SET role='ADMIN' WHERE email='you@example.com';"`.

Включение TLS — это одна переменная окружения (`SITE_ADDRESS=your.domain`) плюс открытие порта 443 — Caddy сам получит и продлит сертификат Let's Encrypt.

---

## 11. Что осознанно вынесено из v1

Следующие функции **не** реализованы. Каждое — намеренное решение из плана, а не забытый пункт.

- **API повышения до ADMIN** — админы заводятся только вне приложения.
- **Самостоятельный сброс пароля** — нет почтовой интеграции.
- **Ротация refresh-токенов** — refresh выдаётся, но эндпоинта обновления нет.
- **Прямые админ-действия с опросами** — нет hide / unhide / delete. Любое скрытие идёт через жалобы и кворум.
- **Смешанный режим опросов** — `options + custom_text` в одном опросе запрещено CHECK-ограничением.
- **Удаление опроса** — нет DELETE. APPROVE-кворум устанавливает `is_hidden = TRUE`.
- **Редактирование/удаление голосов / жалоб / review** — каждый поданный голос и review окончателен.
- **Real-time push** — клиенты перезапрашивают результаты; нет WebSocket / SSE.
- **Видимость по опросу (приватные опросы)** — каждый опрос виден всем аутентифицированным пользователям.
- **Аналитика, time-series, экспорт** — за пределами скоупа.

---

## 12. Карта файлов

```
pollify/
├── api/openapi.yaml              # источник истины для контракта
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/{users,polls,voting,moderation,platform,pkg}/...
│   ├── migrations/000001_init.up.sql
│   ├── tests/integration/api_test.go
│   ├── go.mod, go.sum
│   └── Dockerfile
├── frontend/
│   ├── src/{api,auth,pages}, App.tsx, main.tsx, index.css
│   ├── tests/e2e/{auth,poll-flow,moderation}.spec.ts
│   ├── playwright.config.ts
│   ├── Dockerfile, Caddyfile, vite.config.ts
│   └── package.json
├── docs/
│   ├── PROJECT.md                # английский справочник
│   ├── PROJECT.ru.md             # ← этот файл
│   ├── Pollify-Database-Talk.pptx, Pollify-Database-Talk.ru.pptx
│   ├── system_analytics/         # исходные документы требований
│   └── diagrams/                 # ER- и use-case-диаграммы
├── docker-compose.yml            # dev (backend + postgres)
├── docker-compose.prod.yml       # prod (caddy + api + postgres)
├── Makefile                      # хелперы local-prod
├── DEPLOY.md                     # инструкции развёртывания
├── README.md                     # короткое введение
└── core.md                       # исходные русскоязычные принципы
```

---

## 13. Глоссарий

| Термин | Значение |
| --- | --- |
| **Кворум** | Количество одинаковых решений админов, требуемое для закрытия жалобы. По умолчанию: 2. |
| **Анонимный опрос** | Опрос с `is_anonymous = true`; приложение оставляет `votes.user_id = NULL`; API никогда не отдаёт voter_details. |
| **Участие** | Строка в `poll_participants`, фиксирующая, что пользователь принял участие. Гарантия «один пользователь — один голос» обеспечивается именно здесь, независимо от количества строк в `votes`. |
| **Active / scheduled / completed / hidden** | Вычисляемый статус опроса, выводимый из `start_at`, `end_at`, `is_hidden` и текущего времени. Не хранится. |
| **Self-review** | Админ рассматривает жалобу, которую сам подал. Запрещено и в коде, и триггером БД. |
