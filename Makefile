LIB = libfliptengine
HEADER = flipt_engine.h

.DEFAULT_GOAL := help
.PHONY: build go node python ruby java help clean

# get os and arch
OS := $(shell uname -s | tr A-Z a-z)
ARCH := $(shell uname -m | tr A-Z a-z)

build: ## Build the shared engine library
	cargo build --release

go: build ## Build the go client package
	mkdir -p flipt-client-go/ext/$(OS)_$(ARCH)
	cp target/release/$(LIB).* flipt-client-go/ext/$(OS)_$(ARCH)/
	cp target/release/$(HEADER) flipt-client-go/ext/
	go install ./flipt-client-go

node: build ## Build the node client package
	mkdir -p flipt-client-node/ext/${OS}_${ARCH}
	cp target/release/$(LIB).* flipt-client-node/ext/${OS}_${ARCH}/
	cd flipt-client-node && npm install && npm run build && npm pack

python: build ## Build the python client package
	mkdir -p flipt-client-python/ext/${OS}_${ARCH}
	cp target/release/$(LIB).* flipt-client-python/ext/${OS}_${ARCH}/
	cd flipt-client-python && poetry build

ruby: build ## Build the ruby client gem
	mkdir -p flipt-client-ruby/lib/ext/${OS}_${ARCH}
	cp target/release/$(LIB).* flipt-client-ruby/lib/ext/${OS}_${ARCH}/
	cd flipt-client-ruby && rake install

java: build ## Build the java client package
	mkdir -p flipt-client-java/lib/ext/$(OS)_$(ARCH)
	cp target/release/$(LIB).* flipt-client-java/lib/ext/$(OS)_$(ARCH)/
	cd flipt-client-java && ./gradlew build -x test

clean: ## Clean up build artifacts
	rm -rf target
	rm -rf flipt-client-go/ext/*
	rm -rf flipt-client-node/ext/*
	rm -rf flipt-client-node/*.tgz
	rm -rf flipt-client-node/dist
	rm -rf flipt-client-python/ext/*
	rm -rf flipt-client-python/dist
	rm -rf flipt-client-ruby/lib/ext/*
	rm -rf flipt-client-ruby/pkg/
	rm -rf flipt-client-java/lib

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
