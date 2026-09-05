module order-fill/backend/services/audit-service

go 1.25.5

require (
	order-fill/backend/pkg v0.0.0
	order-fill/backend/proto v0.0.0
)

require (
	github.com/jackc/pgx/v5 v5.10.0
	google.golang.org/grpc v1.83.2
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace order-fill/backend/pkg => ../../pkg

replace order-fill/backend/proto => ../../proto
