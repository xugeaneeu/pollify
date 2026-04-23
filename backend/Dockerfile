FROM golang:1.25.1-alpine AS builder

WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.21

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app/backend

COPY --from=builder /out/api /usr/local/bin/api
COPY backend/migrations ./migrations
COPY api /app/api

USER app

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]
