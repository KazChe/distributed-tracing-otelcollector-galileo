package galileorouterprocessor

import (
	"errors"
)

// Config defines routing rules for mapping standard OTel resource attributes
// to Galileo project/logstream resource attributes.
type Config struct {
	// Routes defines static routing rules. Each route matches resource attributes
	// and injects galileo.project.name and galileo.logstream.name.
	Routes []RouteConfig `mapstructure:"routes"`

	// DefaultRoute defines dynamic fallback routing when no static route matches.
	DefaultRoute DefaultRoute `mapstructure:"default_route"`

	_ struct{}
}

// RouteConfig maps a set of resource attribute key-value pairs to a Galileo project/logstream.
type RouteConfig struct {
	// Match defines resource attribute key-value pairs that must all match for this route to apply.
	Match map[string]string `mapstructure:"match"`

	// Project is the Galileo project name to route matching spans to.
	Project string `mapstructure:"project"`

	// Logstream is the Galileo logstream name to route matching spans to.
	Logstream string `mapstructure:"logstream"`
}

// DefaultRoute defines dynamic routing by deriving Galileo attributes from standard OTel attributes.
type DefaultRoute struct {
	// ProjectFromAttribute is the resource attribute key whose value becomes galileo.project.name.
	ProjectFromAttribute string `mapstructure:"project_from_attribute"`

	// LogstreamFromAttribute is the resource attribute key whose value becomes galileo.logstream.name.
	LogstreamFromAttribute string `mapstructure:"logstream_from_attribute"`

	// LogstreamDefault is used when LogstreamFromAttribute is not found on the resource.
	LogstreamDefault string `mapstructure:"logstream_default"`

	// TODO: support regex matching patterns for route matching
}

func (cfg *Config) Validate() error {
	hasRoutes := len(cfg.Routes) > 0
	hasDefault := cfg.DefaultRoute.ProjectFromAttribute != ""

	if !hasRoutes && !hasDefault {
		return errors.New("at least one route or a default_route with project_from_attribute must be configured")
	}

	for i, route := range cfg.Routes {
		if len(route.Match) == 0 {
			return errors.New("route " + string(rune('0'+i)) + " must have at least one match condition")
		}
		if route.Project == "" {
			return errors.New("route must specify a project")
		}
	}

	return nil
}
