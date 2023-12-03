package raptor

import "github.com/gofiber/fiber/v2"

func wrapHandler(name string, handler func(*Context) error) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("Action", name)
		return handler(&Context{c})
	}
}
