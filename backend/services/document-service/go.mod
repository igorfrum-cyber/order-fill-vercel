module order-fill/backend/services/document-service

go 1.25.5

require (
	order-fill/backend/pkg v0.0.0
	order-fill/backend/proto v0.0.0
)

require (
	github.com/beevik/etree v1.7.1
	github.com/redis/go-redis/v9 v9.22.0
	golang.org/x/sync v0.22.0
	google.golang.org/grpc v1.83.2
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace order-fill/backend/pkg => ../../pkg

replace order-fill/backend/proto => ../../proto
