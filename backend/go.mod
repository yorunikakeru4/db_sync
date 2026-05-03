module backend

go 1.26

require (
	github.com/goccy/go-yaml v1.18.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.6
	github.com/jmoiron/sqlx v1.4.0
	github.com/segmentio/kafka-go v0.4.49
	github.com/stretchr/testify v1.11.1
	github.com/uptrace/bun v1.2.15
	github.com/uptrace/bun/dialect/pgdialect v1.2.15
	github.com/uptrace/bun/driver/pgdriver v1.2.15
	github.com/uptrace/bunrouter v1.0.23
	go.mongodb.org/mongo-driver/v2 v2.4.0
)

replace github.com/cockroachdb/datadriven => ../datadriven
