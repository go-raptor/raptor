package core_test

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/core"
	"github.com/go-raptor/raptor/v4/core/internal/collision"
)

func newTestCore() *core.Core {
	resources := core.NewResources()
	resources.SetLogHandler(slog.NewTextHandler(io.Discard, nil))
	resources.SetConfig(config.NewConfigDefaults())
	return core.NewCore(resources)
}

type DepService struct {
	core.Service

	value string
}

func (s *DepService) Setup() error {
	s.value = "ready"
	return nil
}

type NeedyService struct {
	core.Service

	Dep *DepService

	depInSetup      bool
	depValueInSetup string
}

func (s *NeedyService) Setup() error {
	s.depInSetup = s.Dep != nil
	if s.Dep != nil {
		s.depValueInSetup = s.Dep.value
	}
	return nil
}

func TestServiceSetupSeesInjectedDependencies(t *testing.T) {
	c := newTestCore()
	needy := &NeedyService{}

	err := c.RegisterServices(&core.Components{
		Services: core.Services{&DepService{}, needy},
	})
	if err != nil {
		t.Fatalf("RegisterServices: %v", err)
	}
	if !needy.depInSetup {
		t.Fatal("service Setup ran before dependencies were injected")
	}
	if needy.depValueInSetup != "ready" {
		t.Fatalf("dependency was not set up before dependent Setup ran: got %q", needy.depValueInSetup)
	}
}

type SetupController struct {
	core.Controller

	Dep *DepService

	depInSetup bool
}

func (c *SetupController) Setup() error {
	c.depInSetup = c.Dep != nil
	return nil
}

func TestControllerSetupSeesInjectedDependencies(t *testing.T) {
	c := newTestCore()
	ctrl := &SetupController{}

	if err := c.RegisterServices(&core.Components{Services: core.Services{&DepService{}}}); err != nil {
		t.Fatalf("RegisterServices: %v", err)
	}
	if err := c.RegisterControllers(&core.Components{Controllers: core.Controllers{ctrl}}); err != nil {
		t.Fatalf("RegisterControllers: %v", err)
	}
	if !ctrl.depInSetup {
		t.Fatal("controller Setup ran before dependencies were injected")
	}
}

type SetupMiddleware struct {
	core.Middleware

	Dep *DepService

	depInSetup bool
}

func (m *SetupMiddleware) Setup() error {
	m.depInSetup = m.Dep != nil
	return nil
}

func (m *SetupMiddleware) Handle(ctx *core.Context, next func(*core.Context) error) error {
	return next(ctx)
}

func TestMiddlewareSetupSeesInjectedDependencies(t *testing.T) {
	c := newTestCore()
	mw := &SetupMiddleware{}

	if err := c.RegisterServices(&core.Components{Services: core.Services{&DepService{}}}); err != nil {
		t.Fatalf("RegisterServices: %v", err)
	}
	if err := c.RegisterControllers(&core.Components{}); err != nil {
		t.Fatalf("RegisterControllers: %v", err)
	}
	if err := c.RegisterMiddlewares(&core.Components{Middlewares: core.Middlewares{core.Use(mw)}}); err != nil {
		t.Fatalf("RegisterMiddlewares: %v", err)
	}
	if !mw.depInSetup {
		t.Fatal("middleware Setup ran before dependencies were injected")
	}
}

func TestDuplicateServiceRegistrationFails(t *testing.T) {
	c := newTestCore()

	err := c.RegisterServices(&core.Components{
		Services: core.Services{&DepService{}, &DepService{}},
	})
	if err == nil {
		t.Fatal("registering two services with the same type name should fail")
	}
	if !strings.Contains(err.Error(), "DepService") {
		t.Fatalf("error should name the duplicate service: %v", err)
	}
}

type CollisionService struct {
	core.Service
}

type MismatchController struct {
	core.Controller

	Dep *CollisionService
}

func TestInjectionTypeMismatchFailsCleanly(t *testing.T) {
	c := newTestCore()

	if err := c.RegisterServices(&core.Components{Services: core.Services{&collision.CollisionService{}}}); err != nil {
		t.Fatalf("RegisterServices: %v", err)
	}
	err := c.RegisterControllers(&core.Components{Controllers: core.Controllers{&MismatchController{}}})
	if err == nil {
		t.Fatal("injecting a same-named service of a different type should fail with an error, not succeed")
	}
	if !strings.Contains(err.Error(), "CollisionService") {
		t.Fatalf("error should name the conflicting service: %v", err)
	}
}

type PrivateFieldController struct {
	core.Controller

	dep *DepService
}

func (c *PrivateFieldController) touch() { _ = c.dep }

func TestUnexportedServiceFieldFailsCleanly(t *testing.T) {
	c := newTestCore()

	if err := c.RegisterServices(&core.Components{Services: core.Services{&DepService{}}}); err != nil {
		t.Fatalf("RegisterServices: %v", err)
	}
	err := c.RegisterControllers(&core.Components{Controllers: core.Controllers{&PrivateFieldController{}}})
	if err == nil {
		t.Fatal("injecting into an unexported field should fail with an error, not panic or succeed")
	}
	if !strings.Contains(err.Error(), "exported") {
		t.Fatalf("error should explain the field must be exported: %v", err)
	}
}

type OrphanController struct {
	core.Controller

	Missing *NeedyService
}

func TestMissingServiceDependencyFails(t *testing.T) {
	c := newTestCore()

	err := c.RegisterControllers(&core.Components{Controllers: core.Controllers{&OrphanController{}}})
	if err == nil {
		t.Fatal("depending on an unregistered service should fail at startup")
	}
	if !strings.Contains(err.Error(), "NeedyService") {
		t.Fatalf("error should name the missing service: %v", err)
	}
}
