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
	for scope, data := range routeData {
		currentPath := joinPaths(parentPath, scope)

		switch routeList := data.(type) {
		case []interface{}:
			for _, routeDef := range routeList {
				if route, ok := routeDef.(map[string]interface{}); ok {
					handler, handlerExists := route["handler"]
					if !handlerExists {
						continue
					}

					handlerStr, ok := handler.(string)
					if !ok {
						continue
					}

					for method, pathValue := range route {
						if method == "handler" {
							continue
						}

						pathStr, ok := pathValue.(string)
						if !ok {
							continue
						}

						httpMethod := strings.ToUpper(method)
						if httpMethod == "ANY" {
							httpMethod = "*"
						}

						fullPath := joinPaths(currentPath, pathStr)
						controller, action := core.ParseActionDescriptor(handlerStr)

						*routes = append(*routes, Route{
							Method:     httpMethod,
							Path:       fullPath,
							Controller: core.NormalizeController(controller),
							Action:     action,
						})
					}
				}
			}
		case map[string]interface{}:
			processYAMLRoutes(routeList, currentPath, routes)
		}
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
