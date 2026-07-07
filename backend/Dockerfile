# syntax=docker/dockerfile:1

# --- dev: hot reload via air; source is bind-mounted by docker-compose ---
FROM golang:1.26 AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@v1.65.3
CMD ["air", "-c", ".air.toml"]

# --- build: static production binary ---
FROM golang:1.26 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/api ./cmd/api

# --- prod: minimal runtime ---
FROM gcr.io/distroless/static-debian12:nonroot AS prod
COPY --from=build /out/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
