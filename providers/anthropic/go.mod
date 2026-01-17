module github.com/jllopis/kairos/providers/anthropic

go 1.25

require (
	github.com/anthropics/anthropic-sdk-go v1.0.0
	github.com/jllopis/kairos v0.0.0
)

require (
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
)

replace github.com/jllopis/kairos => ../..
