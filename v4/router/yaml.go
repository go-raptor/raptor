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
	for path, data := range routeData {
		currentPath := normalizePath(joinPaths(parentPath, path))

		switch nested := data.(type) {
		case map[string]interface{}:
			hasHttpMethods := false
			for key := range nested {
				method := strings.ToUpper(key)
				if isHttpMethod(method) {
					hasHttpMethods = true
					break
				}
			}

			if hasHttpMethods {
				for method, handler := range nested {
					httpMethod := strings.ToUpper(method)
					if !isHttpMethod(httpMethod) {
						continue
					}

					if descriptor, ok := handler.(string); ok {
						*routes = append(*routes, MethodRoute(httpMethod, currentPath, descriptor)...)
					}
				}
			} else {
				processYAMLRoutes(nested, currentPath, routes)
			}
		}
	}
}

func joinPaths(parent, child string) string {
	if parent == "" {
		return normalizePath(child)
	}
	if child == "" {
		return normalizePath(parent)
	}
	return normalizePath(parent + "/" + child)
}
