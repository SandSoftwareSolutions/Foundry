# {{PROJECT_NAME}}

A small Go HTTP service by {{AUTHOR}}, using only the standard library.

```console
go run .                      # listens on :8080 (set PORT to change it)
curl localhost:8080/healthz   # {"status":"ok"}
go test ./...
```

It shuts down cleanly on Ctrl-C and has a health endpoint that load balancers and containers can use.
