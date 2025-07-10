module storage-engine

go 1.21

require (
	github.com/spf13/cobra v1.7.0
	google.golang.org/grpc v1.58.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
)

// Additional dependencies to be added:
// github.com/apache/arrow/go/parquet - for Parquet support
// github.com/golang-jwt/jwt/v5 - for JWT authentication  
// github.com/dgraph-io/badger/v4 - for catalog storage
// github.com/prometheus/client_golang - for metrics
