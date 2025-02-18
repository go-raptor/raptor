package router

import (
	"os"
	"strings"

	"github.com/go-raptor/raptor/v3/core"
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
		currentPath := joinPaths(parentPath, path)

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
				for method, handlerValue := range nested {
					httpMethod := strings.ToUpper(method)
					if !isHttpMethod(httpMethod) {
						continue
					}

					handlerStr, ok := handlerValue.(string)
					if !ok {
						continue
					}

					controller, action := core.ParseActionDescriptor(handlerStr)

					*routes = append(*routes, Route{
						Method:     httpMethod,
						Path:       currentPath,
						Controller: core.NormalizeController(controller),
						Action:     action,
					})
				}
			} else {
				processYAMLRoutes(nested, currentPath, routes)
			}
		}
	}
}

func isHttpMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE", "ANY", "*":
		return true
	default:
		return false
	}
}

func joinPaths(parent, child string) string {
	parent = strings.TrimPrefix(parent, "/")
	child = strings.TrimPrefix(child, "/")

	if parent == "" {
		return normalizePath(child)
	}
	if child == "" {
		return normalizePath(parent)
	}
	return normalizePath(parent + "/" + child)
}
