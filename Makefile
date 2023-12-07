LIB = libfliptengine
HEADER = flipt_engine.h

.DEFAULT_GOAL := help
.PHONY: build go node python ruby help clean

build: ## Build the shared engine library
	cargo build --release

go: build ## Prepare the go client for building
	cp target/release/$(LIB).* target/release/$(HEADER) flipt-client-go/

node: build ## Build the node client package
	cp target/release/$(LIB).* flipt-client-node/ext/
	cd flipt-client-node && npm install && npm run build && npm pack

python: build ## Prepare the python client for building
	cp target/release/$(LIB).* flipt-client-python/

ruby: build ## Build the ruby client gem
	cp target/release/$(LIB).* flipt-client-ruby/lib/ext/
	cd flipt-client-ruby && rake install

clean: ## Clean up build artifacts
	rm -rf target
	rm -rf flipt-client-go/$(LIB).* flipt-client-go/$(HEADER)
	rm -rf flipt-client-node/ext/$(LIB).*
	rm -rf flipt-client-node/*.tgz
	rm -rf flipt-client-python/$(LIB).*
	rm -rf flipt-client-ruby/lib/ext/$(LIB).*
	rm -rf flipt-client-ruby/pkg/

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
