package galileorouterprocessor

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const (
	galileoProjectName   = "galileo.project.name"
	galileoLogstreamName = "galileo.logstream.name"
)

type galileoRouter struct {
	config *Config
	logger *zap.Logger
}

func newGalileoRouter(config *Config, logger *zap.Logger) *galileoRouter {
	return &galileoRouter{
		config: config,
		logger: logger,
	}
}

// processTraces iterates over ResourceSpans and injects galileo.project.name
// and galileo.logstream.name resource attributes based on routing rules.
func (r *galileoRouter) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i)
		attrs := rs.Resource().Attributes()

		// Skip if already has Galileo routing attributes — don't overwrite.
		if _, exists := attrs.Get(galileoProjectName); exists {
			continue
		}

		// Try static routes first.
		if r.applyStaticRoute(attrs) {
			continue
		}

		// Fall back to dynamic default route.
		r.applyDefaultRoute(attrs)
	}

	return td, nil
}

// applyStaticRoute checks if any static route matches the resource attributes.
// Returns true if a match was found and attributes were injected.
func (r *galileoRouter) applyStaticRoute(attrs pcommon.Map) bool {
	for _, route := range r.config.Routes {
		if matchesRoute(route, attrs) {
			attrs.PutStr(galileoProjectName, route.Project)
			if route.Logstream != "" {
				attrs.PutStr(galileoLogstreamName, route.Logstream)
			}
			r.logger.Debug("Static route matched",
				zap.String("project", route.Project),
				zap.String("logstream", route.Logstream),
			)
			return true
		}
	}
	return false
}

// applyDefaultRoute derives Galileo attributes from standard OTel resource attributes.
func (r *galileoRouter) applyDefaultRoute(attrs pcommon.Map) {
	def := r.config.DefaultRoute

	if def.ProjectFromAttribute != "" {
		if val, ok := attrs.Get(def.ProjectFromAttribute); ok {
			attrs.PutStr(galileoProjectName, val.Str())
		}
	}

	if def.LogstreamFromAttribute != "" {
		if val, ok := attrs.Get(def.LogstreamFromAttribute); ok {
			attrs.PutStr(galileoLogstreamName, val.Str())
		} else if def.LogstreamDefault != "" {
			attrs.PutStr(galileoLogstreamName, def.LogstreamDefault)
		}
	} else if def.LogstreamDefault != "" {
		attrs.PutStr(galileoLogstreamName, def.LogstreamDefault)
	}
}

// matchesRoute returns true if all key-value pairs in the route's Match map
// are present in the resource attributes with matching values.
func matchesRoute(route RouteConfig, attrs pcommon.Map) bool {
	for key, expected := range route.Match {
		val, ok := attrs.Get(key)
		if !ok || val.Str() != expected {
			return false
		}
	}
	return true
}
