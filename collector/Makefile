BUILDER ?= $(HOME)/go/bin/builder
DIST_DIR = dist

.PHONY: build test clean

build:
	cd $(DIST_DIR) && $(BUILDER) --config=builder-config.yaml

test:
	cd processor/galileorouterprocessor && go test ./...
	cd exporter/galileoexporter && go test ./...

clean:
	rm -rf $(DIST_DIR)/output

docker:
	docker build -t galileo-otel-collector:latest -f Dockerfile ..
