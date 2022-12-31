#!/bin/bash

tinygo build -o www/go.wasm -target wasm www/main.go 
cp "$(tinygo env TINYGOROOT)/targets/wasm_exec.js" www
