FROM golang:1.25-alpine AS builder

RUN apk add --no-cache make git

# Install OTel Collector Builder
RUN go install go.opentelemetry.io/collector/cmd/builder@v0.148.0

WORKDIR /src
COPY . .

# Build the custom collector
RUN cd dist && builder --config=builder-config.yaml

FROM alpine:3.21

COPY --from=builder /src/dist/output/galileo-otel-collector /galileo-otel-collector
ENTRYPOINT ["/galileo-otel-collector"]
CMD ["--config", "/etc/otel/config.yaml"]

EXPOSE 4317 4318 13133
