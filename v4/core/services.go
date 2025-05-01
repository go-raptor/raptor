package core

import (
	"fmt"
	"reflect"

	"github.com/go-raptor/raptor/v4/components"
)

func (c *Core) RegisterServices(components *components.Components) error {
	for _, service := range components.Services {
		serviceName := reflect.TypeOf(service).Elem().Name()
		if err := c.registerService(service, serviceName); err != nil {
			c.Resources.Log.Error("Error while registering service", "service", serviceName, "error", err)
			return err
		}
	}

	for serviceName, service := range c.Services {
		if err := c.injectServices(service, serviceName, "service"); err != nil {
			return err
		}
	}

	return nil
}

func (c *Core) registerService(service components.ServiceInitializer, serviceName string) error {
	if err := c.validateService(service, serviceName); err != nil {
		return err
	}

	if err := service.Init(c.Resources); err != nil {
		c.Resources.Log.Error("Service initialization failed", "service", serviceName, "error", err)
		return err
	}

	c.Services[serviceName] = service

	return nil
}

func (c *Core) validateService(service interface{}, serviceName string) error {
	val := reflect.ValueOf(service)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		err := fmt.Errorf("service must be a non-nil pointer to a struct")
		c.Resources.Log.Error("Error while registering service", "service", serviceName, "error", err)
		return err
	}
	if !val.Type().Implements(reflect.TypeOf((*components.ServiceInitializer)(nil)).Elem()) {
		err := fmt.Errorf("service must implement components.ServiceInitializer")
		c.Resources.Log.Error("Error while registering service", "service", serviceName, "error", err)
		return err
	}
	if val.Elem().FieldByName("Service").Type() != reflect.TypeOf(components.Service{}) {
		err := fmt.Errorf("service must embed components.Service")
		c.Resources.Log.Error("Error while registering service", "service", serviceName, "error", err)
		return err
	}
	return nil
}

func (c *Core) ShutdownServices() error {
	for serviceName, service := range c.Services {
		if err := service.Shutdown(); err != nil {
			c.Resources.Log.Error("Service shutdown failed", "service", serviceName, "error", err)
			return err
		}
	}
	return nil
}

func (c *Core) injectServices(component interface{}, componentName, componentType string) error {
	val := reflect.ValueOf(component)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		err := fmt.Errorf("%s must be a non-nil pointer to a struct", componentType)
		c.Resources.Log.Error(fmt.Sprintf("Error while injecting services into %s", componentType), componentType, componentName, "error", err)
		return err
	}
	val = val.Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if fieldType.Type.Kind() != reflect.Ptr || fieldType.Type.Elem().Kind() != reflect.Struct {
			continue
		}

		if fieldType.Name == "Controller" || fieldType.Name == "Service" {
			continue
		}

		serviceName := fieldType.Type.Elem().Name()
		if service, exists := c.Services[serviceName]; exists {
			if !reflect.TypeOf(service).Implements(reflect.TypeOf((*components.ServiceInitializer)(nil)).Elem()) {
				err := fmt.Errorf("%s.%s expects a service, but %s does not implement ServiceInitializer", componentName, fieldType.Name, serviceName)
				c.Resources.Log.Error(fmt.Sprintf("Error while injecting services into %s", componentType), componentType, componentName, "error", err)
				return err
			}
			field.Set(reflect.ValueOf(service))
			continue
		}

		if fieldType.Type.Implements(reflect.TypeOf((*components.ServiceInitializer)(nil)).Elem()) {
			err := fmt.Errorf("%s requires service %s, but it was not found", componentName, serviceName)
			c.Resources.Log.Error(fmt.Sprintf("Error while injecting services into %s", componentType), componentType, componentName, "error", err)
			return err
		}
	}

	return nil
}
