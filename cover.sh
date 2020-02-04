#!/bin/bash

go test -coverprofile=cover.out ./... || exit 1
go tool cover -html=cover.out
rm -f cover.out
