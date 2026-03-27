package galileoexporter

import (
	"errors"
	"net/url"
	"strings"

	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configretry"
)

const (
	defaultEndpoint  = "https://app.galileo.ai"
	tracesPathSuffix = "/api/galileo/otel/v1/traces"
)

// Config defines configuration for the Galileo exporter.
type Config struct {
	// APIKey is the Galileo API key for authentication.
	// Masked in logs via configopaque.String.
	APIKey configopaque.String `mapstructure:"api_key"`

	// Endpoint is the base URL of the Galileo instance.
	// Defaults to "https://app.galileo.ai".
	// For self-hosted: "https://<your-domain>"
	Endpoint string `mapstructure:"endpoint"`

	// Retry defines retry behavior on transient failures.
	Retry configretry.BackOffConfig `mapstructure:"retry_on_failure"`

	// TODO: fan-out to multiple backends

	_ struct{}
}

// Validate checks that the configuration is valid.
func (cfg *Config) Validate() error {
	if cfg.APIKey == "" {
		return errors.New("api_key is required")
	}
	if cfg.Endpoint != "" {
		if _, err := url.Parse(cfg.Endpoint); err != nil {
			return errors.New("endpoint must be a valid URL")
		}
	}
	return nil
}

// TracesEndpoint returns the full OTLP traces URL for the configured Galileo instance.
func (cfg *Config) TracesEndpoint() string {
	base := cfg.Endpoint
	if base == "" {
		base = defaultEndpoint
	}
	base = strings.TrimRight(base, "/")
	return base + tracesPathSuffix
}
