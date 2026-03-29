package router

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadFromYAML(path string) Routes {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return ParseYAML(data)
}

func ParseYAML(content []byte) Routes {
	var config struct {
		Routes map[string]any `yaml:"routes"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil || config.Routes == nil {
		return nil
	}

	var routes Routes
	parseRoutes(config.Routes, "", &routes)
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
