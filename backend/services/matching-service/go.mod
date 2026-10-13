module order-fill/backend/services/matching-service

go 1.26.9

require (
	order-fill/backend/pkg v0.0.0
	order-fill/backend/proto v0.0.0
)

require google.golang.org/grpc v1.83.2

require (
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace order-fill/backend/pkg => ../../pkg

replace order-fill/backend/proto => ../../proto
