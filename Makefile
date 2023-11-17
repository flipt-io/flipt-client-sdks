LIB = libfliptengine

# run cargo build if anything in engine/ changes
build: $(shell find engine/src/ -type f) ## Build the shared engine library
	cargo build --release
	cd engine && cbindgen --config cbindgen.toml --crate flipt-client-engine --output flipt_engine.h

go: build ## Prepare the go client for building
	cp target/release/$(LIB).* go/flipt-client-go/
	cp engine/flipt_engine.h go/flipt-client-go/

node: build ## Prepare the node client for building
	cp target/release/$(LIB).* node/flipt-client-node/

python: build ## Prepare the python client for building
	cp target/release/$(LIB).* python/flipt-client-python/

ruby: build ## Prepare the ruby client for building
	cp target/release/$(LIB).* ruby/flipt-client-ruby/lib/ext/

.PHONY: go node python ruby help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

