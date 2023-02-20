#!/bin/bash

GOOS="js" GOARCH="ecmascript" gopherjs build -m -o www/cleanplans.js www/main.go 
