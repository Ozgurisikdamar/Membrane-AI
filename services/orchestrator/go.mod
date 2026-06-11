module github.com/Ozgurisikdamar/Membrane-AI/services/orchestrator

go 1.25

require (
	github.com/Ozgurisikdamar/Membrane-AI/pkg v0.0.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/twmb/franz-go v1.18.1
	lukechampine.com/blake3 v1.4.0
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.9.0 // indirect
)

replace github.com/Ozgurisikdamar/Membrane-AI/pkg => ../../pkg
