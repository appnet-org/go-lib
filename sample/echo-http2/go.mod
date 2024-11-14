module github.com/appnet-org/golib/sample/echo-http2

go 1.22.1

replace github.com/appnet-org/golib/sample/echo-pb => ../echo-pb

require (
	github.com/appnet-org/golib/sample/echo-pb v0.0.0-00010101000000-000000000000
	github.com/golang/protobuf v1.5.4
	golang.org/x/net v0.31.0
)

require (
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.66.2 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)
