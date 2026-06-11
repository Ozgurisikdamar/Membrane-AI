module github.com/Ozgurisikdamar/Membrane-AI/services/ingestion

go 1.25.0

require (
	github.com/Ozgurisikdamar/Membrane-AI/pkg v0.0.0
	github.com/Ozgurisikdamar/Membrane-AI/proto v0.0.0
	github.com/google/uuid v1.6.0
	github.com/twmb/franz-go v1.18.1
	google.golang.org/grpc v1.81.1
)

require (
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.9.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/Ozgurisikdamar/Membrane-AI/pkg => ../../pkg

replace github.com/Ozgurisikdamar/Membrane-AI/proto => ../../proto
