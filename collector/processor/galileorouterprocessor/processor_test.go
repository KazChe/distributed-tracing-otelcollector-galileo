package galileorouterprocessor

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func newTestTraces(serviceNames ...string) ptrace.Traces {
	td := ptrace.NewTraces()
	for _, sn := range serviceNames {
		rs := td.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("service.name", sn)
		rs.Resource().Attributes().PutStr("deployment.environment", "production")
		// Add a span so the trace is non-empty.
		ss := rs.ScopeSpans().AppendEmpty()
		span := ss.Spans().AppendEmpty()
		span.SetName("test-span")
	}
	return td
}

func TestStaticRouteMatch(t *testing.T) {
	cfg := &Config{
		Routes: []RouteConfig{
			{
				Match:     map[string]string{"service.name": "agent-a"},
				Project:   "team-alpha",
				Logstream: "prod",
			},
		},
	}
	router := newGalileoRouter(cfg, zap.NewNop())

	td := newTestTraces("agent-a")
	result, err := router.processTraces(context.Background(), td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := result.ResourceSpans().At(0).Resource().Attributes()
	assertAttr(t, attrs, "galileo.project.name", "team-alpha")
	assertAttr(t, attrs, "galileo.logstream.name", "prod")
}

func TestDefaultRouteApplied(t *testing.T) {
	cfg := &Config{
		DefaultRoute: DefaultRoute{
			ProjectFromAttribute:   "service.name",
			LogstreamFromAttribute: "deployment.environment",
			LogstreamDefault:       "default",
		},
	}
	router := newGalileoRouter(cfg, zap.NewNop())

	td := newTestTraces("my-service")
	result, err := router.processTraces(context.Background(), td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := result.ResourceSpans().At(0).Resource().Attributes()
	assertAttr(t, attrs, "galileo.project.name", "my-service")
	assertAttr(t, attrs, "galileo.logstream.name", "production")
}

func TestDefaultRouteFallbackLogstream(t *testing.T) {
	cfg := &Config{
		DefaultRoute: DefaultRoute{
			ProjectFromAttribute:   "service.name",
			LogstreamFromAttribute: "nonexistent.attr",
			LogstreamDefault:       "fallback",
		},
	}
	router := newGalileoRouter(cfg, zap.NewNop())

	td := newTestTraces("my-service")
	result, err := router.processTraces(context.Background(), td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := result.ResourceSpans().At(0).Resource().Attributes()
	assertAttr(t, attrs, "galileo.project.name", "my-service")
	assertAttr(t, attrs, "galileo.logstream.name", "fallback")
}

func TestSkipWhenGalileoAttributeExists(t *testing.T) {
	cfg := &Config{
		Routes: []RouteConfig{
			{
				Match:     map[string]string{"service.name": "agent-a"},
				Project:   "should-not-overwrite",
				Logstream: "should-not-overwrite",
			},
		},
	}
	router := newGalileoRouter(cfg, zap.NewNop())

	td := newTestTraces("agent-a")
	// Pre-set galileo attribute.
	td.ResourceSpans().At(0).Resource().Attributes().PutStr("galileo.project.name", "already-set")

	result, err := router.processTraces(context.Background(), td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := result.ResourceSpans().At(0).Resource().Attributes()
	assertAttr(t, attrs, "galileo.project.name", "already-set")

	// logstream should NOT have been added since we skipped.
	if _, ok := attrs.Get("galileo.logstream.name"); ok {
		t.Error("galileo.logstream.name should not have been set when project already existed")
	}
}

func TestMultipleResourceSpansRoutedIndependently(t *testing.T) {
	cfg := &Config{
		Routes: []RouteConfig{
			{
				Match:     map[string]string{"service.name": "agent-a"},
				Project:   "project-a",
				Logstream: "ls-a",
			},
			{
				Match:     map[string]string{"service.name": "agent-b"},
				Project:   "project-b",
				Logstream: "ls-b",
			},
		},
		DefaultRoute: DefaultRoute{
			ProjectFromAttribute: "service.name",
			LogstreamDefault:     "default",
		},
	}
	router := newGalileoRouter(cfg, zap.NewNop())

	td := newTestTraces("agent-a", "agent-b", "agent-c")
	result, err := router.processTraces(context.Background(), td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// agent-a → static route
	attrs0 := result.ResourceSpans().At(0).Resource().Attributes()
	assertAttr(t, attrs0, "galileo.project.name", "project-a")
	assertAttr(t, attrs0, "galileo.logstream.name", "ls-a")

	// agent-b → static route
	attrs1 := result.ResourceSpans().At(1).Resource().Attributes()
	assertAttr(t, attrs1, "galileo.project.name", "project-b")
	assertAttr(t, attrs1, "galileo.logstream.name", "ls-b")

	// agent-c → default route
	attrs2 := result.ResourceSpans().At(2).Resource().Attributes()
	assertAttr(t, attrs2, "galileo.project.name", "agent-c")
	assertAttr(t, attrs2, "galileo.logstream.name", "default")
}

func TestStaticRoutePriorityOverDefault(t *testing.T) {
	cfg := &Config{
		Routes: []RouteConfig{
			{
				Match:     map[string]string{"service.name": "agent-a"},
				Project:   "static-project",
				Logstream: "static-ls",
			},
		},
		DefaultRoute: DefaultRoute{
			ProjectFromAttribute:   "service.name",
			LogstreamFromAttribute: "deployment.environment",
		},
	}
	router := newGalileoRouter(cfg, zap.NewNop())

	td := newTestTraces("agent-a")
	result, err := router.processTraces(context.Background(), td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attrs := result.ResourceSpans().At(0).Resource().Attributes()
	assertAttr(t, attrs, "galileo.project.name", "static-project")
	assertAttr(t, attrs, "galileo.logstream.name", "static-ls")
}

func TestConfigValidation(t *testing.T) {
	// No routes and no default → error
	cfg := &Config{}
	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for empty config")
	}

	// Route with no match → error
	cfg = &Config{
		Routes: []RouteConfig{{Project: "p"}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for route with empty match")
	}

	// Route with no project → error
	cfg = &Config{
		Routes: []RouteConfig{{Match: map[string]string{"k": "v"}}},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for route with empty project")
	}

	// Valid config with only default route
	cfg = &Config{
		DefaultRoute: DefaultRoute{ProjectFromAttribute: "service.name"},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}

	// Valid config with routes
	cfg = &Config{
		Routes: []RouteConfig{
			{Match: map[string]string{"service.name": "x"}, Project: "p"},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

// assertAttr checks that a resource attribute exists with the expected string value.
func assertAttr(t *testing.T, attrs pcommon.Map, key, expected string) {
	t.Helper()
	val, ok := attrs.Get(key)
	if !ok {
		t.Errorf("expected attribute %q to exist", key)
		return
	}
	if val.Str() != expected {
		t.Errorf("attribute %q: got %q, want %q", key, val.Str(), expected)
	}
}
