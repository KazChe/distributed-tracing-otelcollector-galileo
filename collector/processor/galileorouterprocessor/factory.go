package galileorouterprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/KazChe/distributed-tracing-otelcollector-galileo/collector/processor/galileorouterprocessor/internal/metadata"
)

// NewFactory returns a new factory for the galileo_router processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(createTraces, metadata.TracesStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		DefaultRoute: DefaultRoute{
			ProjectFromAttribute:   "service.name",
			LogstreamFromAttribute: "deployment.environment",
			LogstreamDefault:       "default",
		},
	}
}

func createTraces(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	processorCfg := cfg.(*Config)
	router := newGalileoRouter(processorCfg, set.Logger)

	return processorhelper.NewTraces(
		ctx, set, cfg, nextConsumer,
		router.processTraces,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}
