FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ai ./cmd/ai

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=build /out/ai /app/ai-server

ENV AI_HOST=0.0.0.0
ENV AI_PORT=18083

EXPOSE 18083

ENTRYPOINT ["/app/ai-server"]
