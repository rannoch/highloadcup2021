GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=hlcup2021
OPENAPIGENERATOR=openapi-generator

all: test lint

test:
	$(GOTEST) ./... -v -coverprofile=coverage.txt -covermode=atomic

build:
	$(GOBUILD) -o $(BINARY_NAME)

generate:
	$(OPENAPIGENERATOR) generate -g go --global-property models --additional-properties=isGoSubmodule=false,packageName=model -i swagger.yaml -o models