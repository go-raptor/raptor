package router

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadFromYAML(path string) Routes {
	data, err := os.ReadFile(path)
	if err != nil {
		return Routes{}
	}

	return ParseYAML(data)
}

func ParseYAML(content []byte) Routes {
	var yamlConfig struct {
		Routes map[string]interface{} `yaml:"routes"`
	}

	if err := yaml.Unmarshal(content, &yamlConfig); err != nil {
		return Routes{}
	}

	if yamlConfig.Routes == nil {
		return Routes{}
	}

	var routes Routes
	processYAMLRoutes(yamlConfig.Routes, "", &routes)
	return routes
}

func processYAMLRoutes(routeData map[string]interface{}, parentPath string, routes *Routes) {
	for key, data := range routeData {
		upperKey := strings.ToUpper(key)
		if isHttpMethod(upperKey) {
			if descriptor, ok := data.(string); ok {
				*routes = append(*routes, MethodRoute(upperKey, parentPath, descriptor)...)
			}
			continue
		}

		currentPath := joinPaths(parentPath, key)

		switch nested := data.(type) {
		case map[string]interface{}:
			processYAMLRoutes(nested, currentPath, routes)
		case string:
			*routes = append(*routes, MethodRoute("ANY", currentPath, nested)...)
		}
	}
}

func joinPaths(parent, child string) string {
	return normalizePath(parent + "/" + child)
}
