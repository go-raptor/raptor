package core

import (
	"fmt"
	"reflect"
)

var (
	serviceType            = reflect.TypeFor[Service]()
	serviceInitializerType = reflect.TypeFor[ServiceInitializer]()
)

type Services []ServiceInitializer

type ServiceInitializer interface {
	Init(*Resources) error
	Shutdown() error
}

type Service struct {
	*Resources
	onInit     func() error
	onShutdown func() error
}

func (s *Service) Init(resources *Resources) error {
	s.Resources = resources
	if s.onInit != nil {
		return s.onInit()
	}
	return nil
}

func (s *Service) Shutdown() error {
	if s.onShutdown != nil {
		return s.onShutdown()
	}
	return nil
}

func (s *Service) OnInit(callback func() error) {
	s.onInit = callback
}

func (s *Service) OnShutdown(callback func() error) {
	s.onShutdown = callback
}

func (c *Core) RegisterServices(components *Components) error {
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

func (c *Core) registerService(service ServiceInitializer, serviceName string) error {
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

func (c *Core) validateService(service any, serviceName string) error {
	val := reflect.ValueOf(service)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		err := fmt.Errorf("service must be a non-nil pointer to a struct")
		c.Resources.Log.Error("Error while registering service", "service", serviceName, "error", err)
		return err
	}

	field := val.Elem().FieldByName("Service")
	if !field.IsValid() || field.Type() != serviceType {
		err := fmt.Errorf("service must embed raptor.Service")
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

func (c *Core) injectServices(component any, componentName, componentType string) error {
	val := reflect.ValueOf(component).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if fieldType.Type.Kind() != reflect.Pointer || fieldType.Type.Elem().Kind() != reflect.Struct {
			continue
		}

		if fieldType.Name == "Controller" || fieldType.Name == "Service" || fieldType.Name == "Middleware" {
			continue
		}

		serviceName := fieldType.Type.Elem().Name()
		if service, exists := c.Services[serviceName]; exists {
			field.Set(reflect.ValueOf(service))
			continue
		}

		if fieldType.Type.Implements(serviceInitializerType) {
			err := fmt.Errorf("%s requires service %s, but it was not found", componentName, serviceName)
			c.Resources.Log.Error(fmt.Sprintf("Error while injecting services into %s", componentType), componentType, componentName, "error", err)
			return err
		}
	}

	return nil
}
