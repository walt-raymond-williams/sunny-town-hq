FROM node:22-alpine AS frontend-build

WORKDIR /src/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend ./
RUN npm run build

FROM golang:1.22-alpine AS backend-build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/hq ./cmd/hq

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=backend-build /out/hq /app/hq-server
COPY --from=frontend-build /src/web /app/web
COPY deploy/postgres/migrations /app/deploy/postgres/migrations

ENV HQ_HOST=0.0.0.0
ENV HQ_PORT=18080

EXPOSE 18080

ENTRYPOINT ["/app/hq-server"]
