package core

import (
	"fmt"
	"reflect"

	"github.com/go-raptor/components"
)

func (c *Core) RegisterServices(components *Components) error {
	for _, service := range components.Services {
		if err := service.InitService(c.utils); err != nil {
			c.utils.Log.Error("Service initialization failed", "service", reflect.TypeOf(service).Elem().Name(), "error", err)
			return err
		}
		c.services[reflect.TypeOf(service).Elem().Name()] = service
	}

	for _, service := range c.services {
		if err := c.injectServicesToService(service); err != nil {
			return err
		}
	}

	return nil
}

func (c *Core) ShutdownServices() {
	for _, service := range c.services {
		if err := service.ShutdownService(); err != nil {
			c.utils.Log.Error("Service shutdown failed", "service", reflect.TypeOf(service).Elem().Name(), "error", err)
		}
	}
}

func (c *Core) injectServicesToController(controllerValue reflect.Value, controller string) error {
	controllerElem := controllerValue.Elem()

	for i := 0; i < controllerElem.NumField(); i++ {
		field := controllerElem.Field(i)
		fieldType := controllerElem.Type().Field(i)

		if fieldType.Type.Kind() != reflect.Ptr ||
			fieldType.Type.Elem().Kind() != reflect.Struct {
			continue
		}

		if fieldType.Name == "Controller" {
			continue
		}

		service := fieldType.Type.Elem().Name()
		if injectedService, ok := c.services[service]; ok {
			field.Set(reflect.ValueOf(injectedService))
			continue
		}

		serviceInterfaceType := reflect.TypeOf((*components.ServiceProvider)(nil)).Elem()
		if fieldType.Type.Implements(serviceInterfaceType) {
			return fmt.Errorf("%s requires %s, but the service was not found in services initializer", controller, service)
		}
	}

	return nil
}

func (c *Core) injectServicesToService(service components.ServiceProvider) error {
	serviceValue := reflect.ValueOf(service).Elem()
	serviceType := reflect.TypeOf(service).Elem()

	for i := 0; i < serviceValue.NumField(); i++ {
		field := serviceValue.Field(i)
		fieldType := serviceType.Field(i)

		if fieldType.Type.Kind() != reflect.Ptr || fieldType.Type.Elem().Kind() != reflect.Struct {
			continue
		}

		if injectedService, ok := c.services[fieldType.Type.Elem().Name()]; ok {
			field.Set(reflect.ValueOf(injectedService))
			continue
		}

		serviceInterfaceType := reflect.TypeOf((*components.ServiceProvider)(nil)).Elem()
		if fieldType.Type.Implements(serviceInterfaceType) {
			err := fmt.Errorf("%s requires %s, but the service was not found in services initializer", serviceType.Name(), fieldType.Type.Elem().Name())
			c.utils.Log.Error("Error while registering service", "service", serviceType.Name(), "error", err)
			return err
		}
	}

	return nil
}

func (c *Core) injectServicesToMiddleware(middleware components.MiddlewareProvider) error {
	middlewareValue := reflect.ValueOf(middleware).Elem()
	middlewareType := reflect.TypeOf(middleware).Elem()

	for i := 0; i < middlewareValue.NumField(); i++ {
		field := middlewareValue.Field(i)
		fieldType := middlewareType.Field(i)

		if fieldType.Type.Kind() != reflect.Ptr || fieldType.Type.Elem().Kind() != reflect.Struct {
			continue
		}

		serviceName := fieldType.Type.Elem().Name()
		if injectedService, ok := c.services[serviceName]; ok {
			field.Set(reflect.ValueOf(injectedService))
			continue
		}

		serviceInterfaceType := reflect.TypeOf((*components.ServiceProvider)(nil)).Elem()
		if fieldType.Type.Implements(serviceInterfaceType) {
			err := fmt.Errorf("%s requires %s, but the service was not found in services initializer", middlewareType.Name(), serviceName)
			c.utils.Log.Error("Error while registering middleware", "middleware", middlewareType.Name(), "error", err)
			return err
		}
	}

	return nil
}
