package raptor

import "github.com/gofiber/fiber/v3"

func wrapHandler(handler func(*Context) error) fiber.Handler {
	return func(c fiber.Ctx) error {
		return handler(&Context{c.(*fiber.DefaultCtx), "", ""})
	}
}

func wrapActionHandler(controller, action string, handler func(*Context) error) fiber.Handler {
	return func(c fiber.Ctx) error {
		return handler(&Context{c.(*fiber.DefaultCtx), controller, action})
	}
}
