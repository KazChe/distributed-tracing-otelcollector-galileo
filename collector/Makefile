BUILDER ?= $(HOME)/go/bin/builder
DIST_DIR = dist

.PHONY: build test clean

build:
	cd $(DIST_DIR) && $(BUILDER) --config=builder-config.yaml

test:
	go test ./processor/galileorouterprocessor/...
	go test ./exporter/galileoexporter/...

clean:
	rm -rf $(DIST_DIR)/output

docker:
	docker build -t galileo-otel-collector:latest -f Dockerfile ..
