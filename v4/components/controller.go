package components

import "strings"

const controllerSuffix = "Controller"
const descriptorSeparator = "."

func NormalizeController(controller string) string {
	if !strings.HasSuffix(controller, controllerSuffix) {
		return controller + controllerSuffix
	}
	return controller
}

func ParseActionDescriptor(descriptor string) (controller, action string) {
	parts := strings.Split(descriptor, descriptorSeparator)
	if len(parts) == 2 {
		return NormalizeController(parts[0]), parts[1]
	}
	return NormalizeController(descriptor), ""
}
