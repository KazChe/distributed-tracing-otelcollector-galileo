package galileoexporter

import (
	"testing"

	"go.opentelemetry.io/collector/config/configopaque"
)

func TestConfigValidation(t *testing.T) {
	// Missing API key.
	cfg := &Config{Endpoint: "https://example.com"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing api_key")
	}

	// Valid config.
	cfg = &Config{
		APIKey:   configopaque.String("test-key"),
		Endpoint: "https://example.com",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}

	// Empty endpoint is valid (uses default).
	cfg = &Config{
		APIKey: configopaque.String("test-key"),
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestTracesEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{
			name:     "default endpoint",
			endpoint: "",
			want:     "https://app.galileo.ai/api/galileo/otel/v1/traces",
		},
		{
			name:     "custom endpoint",
			endpoint: "https://my-galileo.example.com",
			want:     "https://my-galileo.example.com/api/galileo/otel/v1/traces",
		},
		{
			name:     "trailing slash stripped",
			endpoint: "https://my-galileo.example.com/",
			want:     "https://my-galileo.example.com/api/galileo/otel/v1/traces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Endpoint: tt.endpoint}
			got := cfg.TracesEndpoint()
			if got != tt.want {
				t.Errorf("TracesEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIKeyMasked(t *testing.T) {
	key := configopaque.String("super-secret-key")
	// configopaque.String.String() should return "[REDACTED]"
	if key.String() != "[REDACTED]" {
		t.Errorf("expected API key to be masked, got %q", key.String())
	}
}
