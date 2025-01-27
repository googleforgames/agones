module agones.dev/agones/examples/simple-game-server

go 1.23

require agones.dev/agones v1.36.0

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace agones.dev/agones => ../../
