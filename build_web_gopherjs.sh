#!/bin/bash

GOOS="js" GOARCH="ecmascript" gopherjs build -o ./gopherjs_www/main.js ./gopherjs_www/main.go 
