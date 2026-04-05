FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
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
