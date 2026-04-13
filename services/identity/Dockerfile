FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
# Install sqlc for code generation
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Generate sqlc code
RUN cd services/identity/infrastructure/postgres/sqlc && sqlc generate
RUN cd services/org/infrastructure/postgres/sqlc && sqlc generate
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /identity \
    ./services/identity/cmd/identity

FROM gcr.io/distroless/static-debian12
COPY --from=builder /identity /identity
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/identity"]
