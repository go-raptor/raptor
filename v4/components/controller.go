package components

import "strings"

const controllerSuffix = "Controller"
const descriptorSeparator = "."

type Controllers []ControllerInitializer

type ControllerInitializer interface {
	Init(*Resources)
}

type Controller struct {
	*Resources
	onInit func()
}

func (c *Controller) Init(resources *Resources) {
	c.Resources = resources
	if c.onInit != nil {
		c.onInit()
	}
}

func (c *Controller) OnInit(callback func()) {
	c.onInit = callback
}

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

func ActionDescriptor(controller, action string) string {
	return controller + descriptorSeparator + action
}
