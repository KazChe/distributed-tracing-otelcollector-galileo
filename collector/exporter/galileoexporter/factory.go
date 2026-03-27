package galileoexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/KazChe/distributed-tracing-otelcollector-galileo/collector/exporter/galileoexporter/internal/metadata"
)

// NewFactory returns a new factory for the Galileo exporter.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		metadata.Type,
		createDefaultConfig,
		exporter.WithTraces(createTraces, metadata.TracesStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Endpoint: defaultEndpoint,
		Retry:    configretry.NewDefaultBackOffConfig(),
	}
}

func createTraces(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
) (exporter.Traces, error) {
	galileoCfg := cfg.(*Config)

	exp, err := newGalileoExporter(galileoCfg, set)
	if err != nil {
		return nil, err
	}

	return exporterhelper.NewTraces(
		ctx, set, cfg,
		exp.pushTraces,
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}), // HTTP client handles timeout
		exporterhelper.WithRetry(galileoCfg.Retry),
	)
}
