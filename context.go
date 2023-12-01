package raptor

import "github.com/gofiber/fiber/v2"

type Map map[string]interface{}

type Context struct {
	*fiber.Ctx
}

func (c *Context) JSON(status int, data interface{}) error {
	return c.Ctx.Status(status).JSON(data)
}
