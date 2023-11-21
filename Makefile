LIB = libfliptengine

.DEFAULT_GOAL := help
.PHONY: go node python ruby help clean

# run cargo build if anything in engine/ changes
build: ## Build the shared engine library
	cargo build --release

go: build ## Prepare the go client for building
	cp target/release/$(LIB).* flipt-client-go/
	cp target/release/flipt_engine.h flipt-client-go/

node: build ## Prepare the node client for building
	cp target/release/$(LIB).* flipt-client-node/

python: build ## Prepare the python client for building
	cp target/release/$(LIB).* flipt-client-python/

ruby: build ## Prepare the ruby client for building
	cp target/release/$(LIB).* flipt-client-ruby/lib/ext/

clean: ## Clean up build artifacts
	rm -rf target
	rm -rf flipt-client-go/$(LIB).*
	rm -rf flipt-client-node/$(LIB).*
	rm -rf flipt-client-python/$(LIB).*
	rm -rf flipt-client-ruby/lib/ext/$(LIB).*
	rm -rf flipt-client-ruby/pkg/

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
