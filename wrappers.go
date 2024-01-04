package raptor

import "github.com/gofiber/fiber/v2"

func wrapHandler(handler func(*Context) error) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return handler(&Context{c})
	}
}

func wrapActionHandler(name string, handler func(*Context) error) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("Action", name)
		return handler(&Context{c})
	}
}
