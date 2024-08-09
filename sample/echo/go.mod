module github.com/appnet-org/golib/sample/echo

go 1.22.1

// toolchain go1.22.2

require (
	github.com/appnet-org/golib/interceptor v0.0.0-00010101000000-000000000000
	github.com/appnet-org/golib/sample/echo-pb v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.24.0
	google.golang.org/grpc v1.63.2
)

require (
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240227224415-6ceb2ff114de // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/appnet-org/golib/sample/echo-pb => ../echo-pb

replace github.com/appnet-org/golib/sample/echo/roundrobinlb => ./roundrobinlb

replace github.com/appnet-org/golib/interceptor => ../../interceptor
