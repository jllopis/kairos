module github.com/jllopis/kairos/examples/16-providers

go 1.25

require (
	github.com/jllopis/kairos v0.0.0
	github.com/jllopis/kairos/providers/anthropic v0.0.0
	github.com/jllopis/kairos/providers/gemini v0.0.0
	github.com/jllopis/kairos/providers/openai v0.0.0
	github.com/jllopis/kairos/providers/qwen v0.0.0
)

require (
	cloud.google.com/go v0.116.0 // indirect
	cloud.google.com/go/auth v0.9.3 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	github.com/anthropics/anthropic-sdk-go v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/openai/openai-go v1.0.0 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.44.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genai v1.0.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/grpc v1.77.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace (
	github.com/jllopis/kairos => ../..
	github.com/jllopis/kairos/providers/anthropic => ../../providers/anthropic
	github.com/jllopis/kairos/providers/gemini => ../../providers/gemini
	github.com/jllopis/kairos/providers/openai => ../../providers/openai
	github.com/jllopis/kairos/providers/qwen => ../../providers/qwen
)
