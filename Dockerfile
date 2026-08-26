# ---- Stage 1: build del frontend (Tailwind + vendor + fuentes) ----
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# ---- Stage 2: build del backend Go (estático, sin CGO) ----
FROM golang:1.22-alpine AS go
WORKDIR /src
# Cache de módulos
COPY go.mod go.sum ./
RUN go mod download
# Código fuente
COPY . .
# Trae el frontend ya compilado para embebirlo
COPY --from=web /web/dist ./web/dist
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -ldflags="-s -w" -o /out/server .

# ---- Stage 3: runtime mínimo (scratch) ----
FROM scratch
WORKDIR /app
# Certificados para TLS saliente (por si se usan webhooks externos)
COPY --from=go /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=go /out/server /app/server
ENV ADDR=:8080
ENV DATABASE_PATH=/data/tasks.db
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/server"]
