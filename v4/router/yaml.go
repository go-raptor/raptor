package router

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadRoutesFromYAML reads and parses a routes YAML file.
func LoadRoutesFromYAML(path string) (Routes, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read routes file %s: %w", path, err)
	}
	return ParseRoutesYAML(data)
}

// ParseRoutesYAML parses a routes YAML document.
func ParseRoutesYAML(content []byte) (Routes, error) {
	var config struct {
		Routes map[string]any `yaml:"routes"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("parse routes YAML: %w", err)
	}
	if config.Routes == nil {
		return nil, fmt.Errorf("routes YAML: missing 'routes' key")
	}

	var routes Routes
	parseRoutes(config.Routes, "", &routes)
	return routes, nil
}

// LoadFromYAML reads routes from a YAML file, returning nil on any error.
//
// Deprecated: use LoadRoutesFromYAML for proper error handling.
func LoadFromYAML(path string) Routes {
	routes, _ := LoadRoutesFromYAML(path)
	return routes
}

// ParseYAML parses a routes YAML document, returning nil on any error.
//
// Deprecated: use ParseRoutesYAML for proper error handling.
func ParseYAML(content []byte) Routes {
	routes, _ := ParseRoutesYAML(content)
	return routes
}

func parseRoutes(data map[string]any, parentPath string, routes *Routes) {
	for key, value := range data {
		if upper := strings.ToUpper(key); isHTTPMethod(upper) {
			if descriptor, ok := value.(string); ok {
				*routes = append(*routes, MethodRoute(upper, parentPath, descriptor)...)
			}
			continue
		}

		path := normalizePath(parentPath + "/" + key)

		switch v := value.(type) {
		case map[string]any:
			parseRoutes(v, path, routes)
		case string:
			*routes = append(*routes, MethodRoute("ANY", path, v)...)
		}
	}
}
