FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN cd services/identity/infrastructure/postgres/sqlc && sqlc generate
RUN cd services/org/infrastructure/postgres/sqlc && sqlc generate
RUN cd services/notification/infrastructure/postgres/sqlc && sqlc generate
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /notification \
    ./services/notification/cmd/notification

FROM gcr.io/distroless/static-debian12
COPY --from=builder /notification /notification
USER nonroot:nonroot
EXPOSE 8083
ENTRYPOINT ["/notification"]
