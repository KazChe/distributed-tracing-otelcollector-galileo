package galileoexporter

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.uber.org/zap"
)

type galileoExporter struct {
	config    *Config
	logger    *zap.Logger
	client    *http.Client
	tracesURL string
}

func newGalileoExporter(cfg *Config, set exporter.Settings) (*galileoExporter, error) {
	return &galileoExporter{
		config:    cfg,
		logger:    set.Logger,
		tracesURL: cfg.TracesEndpoint(),
	}, nil
}

// start is called when the exporter starts. Initializes the HTTP client.
func (e *galileoExporter) start(_ context.Context, _ component.Host) error {
	e.client = &http.Client{
		Timeout: 30_000_000_000, // 30 seconds in nanoseconds
	}
	e.logger.Info("Galileo exporter started",
		zap.String("endpoint", e.tracesURL),
	)
	return nil
}

// pushTraces serializes traces as protobuf and sends them to the Galileo OTLP endpoint.
func (e *galileoExporter) pushTraces(ctx context.Context, td ptrace.Traces) error {
	// Marshal traces to OTLP protobuf.
	request := ptraceotlp.NewExportRequestFromTraces(td)
	body, err := request.MarshalProto()
	if err != nil {
		return consumererror.NewPermanent(fmt.Errorf("failed to marshal traces: %w", err))
	}

	// Gzip compress the body.
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	if _, err := gz.Write(body); err != nil {
		return consumererror.NewPermanent(fmt.Errorf("failed to compress traces: %w", err))
	}
	if err := gz.Close(); err != nil {
		return consumererror.NewPermanent(fmt.Errorf("failed to close gzip writer: %w", err))
	}

	// Build HTTP request.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.tracesURL, &compressed)
	if err != nil {
		return consumererror.NewPermanent(fmt.Errorf("failed to create HTTP request: %w", err))
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Galileo-API-Key", string(e.config.APIKey))

	// Send.
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send traces to Galileo: %w", err)
	}
	defer resp.Body.Close()

	// Drain body to allow connection reuse.
	_, _ = io.ReadAll(resp.Body)

	// Handle response status.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Retryable errors.
	if resp.StatusCode == 429 || resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
		return fmt.Errorf("Galileo returned retryable status %d", resp.StatusCode)
	}

	// Permanent errors.
	return consumererror.NewPermanent(fmt.Errorf("Galileo returned status %d", resp.StatusCode))
}
