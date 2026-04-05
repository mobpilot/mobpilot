module github.com/knobo/driftbase

go 1.23

require (
	connectrpc.com/connect v1.17.0
	github.com/go-chi/chi/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.6.0
	github.com/jackc/pgx/v5 v5.7.2
	github.com/knadh/koanf/v2 v2.1.2
	github.com/knadh/koanf/providers/env v1.0.0
	github.com/nats-io/nats.go v1.39.0
	github.com/ory/hydra-client-go/v2 v2.2.1
	github.com/ory/kratos-client-go v1.3.8
	github.com/ory/keto-client-go v0.13.0-alpha.0
	github.com/riverqueue/river v0.14.2
	github.com/riverqueue/river/riverdriver/riverpgxv5 v0.14.2
	go.opentelemetry.io/otel v1.34.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.34.0
	go.opentelemetry.io/otel/sdk v1.34.0
	go.opentelemetry.io/otel/trace v1.34.0
	golang.org/x/net v0.35.0
	google.golang.org/protobuf v1.36.5
)
