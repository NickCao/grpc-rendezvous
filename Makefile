proto/rendezvous.pb.go proto/rendezvous_grpc.pb.go: proto/rendezvous.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/rendezvous.proto
