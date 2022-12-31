#!/bin/bash

GOOS=windows GOARCH=amd64 go build -o inkscape_extension/cleanplans/cleanplans.exe
GOOS=linux GOARCH=amd64 go build -o inkscape_extension/cleanplans/cleanplans_linux
GOOS=darwin GOARCH=amd64 go build -o inkscape_extension/cleanplans/cleanplans_osx

pushd inkscape_extension
rm cleanplans.zip
zip -r cleanplans.zip .
popd
