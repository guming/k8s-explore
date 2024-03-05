CUR_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

VERSION=v0.0.0
GIT_COMMIT=$(shell git rev-parse --verify HEAD)
UTC_NOW=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

.PHONY: back-run-dev
back-run-dev:
	go run ${CUR_DIR}/main.go --host 0.0.0.0 --port 8090