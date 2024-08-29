```
go build -buildmode=plugin -o /tmp/appnet/interceptors/lb20240828172309 .
```

change `grpc.WithUnaryInterceptor(interceptor.ClientInterceptor("/interceptors/frontend", "/interceptors/lb")),` to `grpc.WithUnaryInterceptor(interceptor.ClientInterceptor("/interceptors/frontend", "/tmp/appnet/interceptors/lb")),`