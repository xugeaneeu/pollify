# Deployment

Single-host Docker deploy. Postgres + API stay on the internal compose
network; Caddy fronts both, serving the built React SPA at `/` and
proxying `/api/*` and `/healthz` to the API. Plain HTTP on port 80 by
default; flip on TLS by changing one env var (see "Adding TLS" below).

## Prerequisites on the server

A reasonably recent Linux box with:

* `docker` (v20.10+) and `docker compose` (v2)
* Ports 80 (and later 443) open to the world
* SSH access for you

A 1-vCPU / 1 GB RAM VPS is enough for a small deployment; the api
binary is ~12 MB and Caddy + Postgres are similarly modest.

## First deploy

```bash
# on your laptop
ssh you@your-server

# on the server
sudo mkdir -p /srv && sudo chown $USER /srv && cd /srv
git clone https://github.com/<you>/pollify.git
cd pollify

# secrets
cp .env.example .env
chmod 600 .env
$EDITOR .env   # set POSTGRES_PASSWORD and JWT_SECRET (see hints in the file)

# build + start everything
docker compose -f docker-compose.prod.yml up -d --build

# tail the logs while the api applies migrations
docker compose -f docker-compose.prod.yml logs -f api
```

Once the api logs `starting api server`, hit `http://<SERVER_IP>` from
your browser. You should land on the login screen.

## Promoting the first admin

`POST /auth/register` always creates a `USER`. To get an admin (needed
for the moderation queue), create your account through the UI, then on
the server:

```bash
docker compose -f docker-compose.prod.yml exec postgres \
  psql -U pollify -d pollify \
  -c "UPDATE users SET role='ADMIN' WHERE email='you@example.com';"
```

Sign out and back in to refresh the JWT — the **Moderation** link
appears in the navbar. Quorum is 2, so promote at least one more
account if you actually want to resolve reports.

## Redeploying

```bash
ssh you@your-server
cd /srv/pollify
git pull
docker compose -f docker-compose.prod.yml up -d --build
```

`up -d --build` rebuilds only the images whose source changed and
swaps in the new containers. Migrations run on api startup; the
Postgres data volume is preserved across redeploys.

## Day-to-day operations

| Task | Command |
| --- | --- |
| Live logs (all services) | `docker compose -f docker-compose.prod.yml logs -f` |
| Live logs (one service) | `docker compose -f docker-compose.prod.yml logs -f api` |
| psql shell | `docker compose -f docker-compose.prod.yml exec postgres psql -U pollify -d pollify` |
| Restart api only | `docker compose -f docker-compose.prod.yml restart api` |
| Stop everything | `docker compose -f docker-compose.prod.yml down` |
| Stop + wipe DB volume | `docker compose -f docker-compose.prod.yml down -v` ⚠️ |

### Backups

The Postgres data volume is `pollify-prod_postgres-data` (compose
prefixes it with the project name from `name:` in the compose file).
A simple cron-friendly dump:

```bash
docker compose -f docker-compose.prod.yml exec -T postgres \
  pg_dump -U pollify -d pollify -Fc > /srv/backups/pollify-$(date +%F).dump
```

Restore by piping `pg_restore` into the same container.

## Adding TLS later

Caddy ships with automatic Let's Encrypt. When you have a domain
pointing at the server:

1. In `.env`, set `SITE_ADDRESS=pollify.example.com` (no scheme, no
   port).
2. In `docker-compose.prod.yml`, add `- "443:443"` to `web.ports`.
3. Open port 443 in your firewall.
4. `docker compose -f docker-compose.prod.yml up -d`

Caddy will obtain and renew the cert on its own. No other changes
needed — the SPA already uses relative `/api/*` paths.

## Troubleshooting

**"set POSTGRES_PASSWORD in .env" / "set JWT_SECRET in .env" on `up`**
Compose refuses to start without those values. Fill them in `.env`.

**`web` shows blank page or 502 on `/api/*`**
The api container probably failed to apply migrations or connect to
postgres. Check `docker compose ... logs api` — the most common cause
is a wrong `POSTGRES_PASSWORD` between the api and postgres services.
They both read from `.env`, so a fresh `up -d` after fixing `.env`
clears it.

**Migrations changed and failed to apply**
The api logs the failing migration filename. Roll back with
`platformpg.RollbackMigrations` (manual) or restore from backup.

**Reset everything (⚠️ destroys data)**
```bash
docker compose -f docker-compose.prod.yml down -v
docker compose -f docker-compose.prod.yml up -d --build
```
