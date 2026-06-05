FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sunny-town ./cmd/sunny-town

FROM alpine:3.20

WORKDIR /app

COPY --from=build /out/sunny-town /app/sunny-town-server
COPY sunny-town/maps /app/sunny-town/maps

ENV SUNNY_TOWN_HOST=0.0.0.0
ENV SUNNY_TOWN_PORT=18082
ENV SUNNY_TOWN_MAPS_DIR=/app/sunny-town/maps

EXPOSE 18082

ENTRYPOINT ["/app/sunny-town-server"]
