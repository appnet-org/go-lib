module github.com/appnet-org/golib/sample/echo-plain

go 1.22.1

// toolchain go1.22.2

require (
	github.com/appnet-org/golib/sample/echo-pb v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.29.0
	google.golang.org/grpc v1.66.2
)

require (
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/appnet-org/golib/sample/echo-pb => ../echo-pb
