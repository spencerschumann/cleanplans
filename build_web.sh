#!/bin/bash

GOOS="js" GOARCH="wasm" go build -tags float32 -o www/go.wasm www/main.go
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" www
