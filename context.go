package raptor

import "github.com/gofiber/fiber/v2"

type Map map[string]interface{}

type Context struct {
	*fiber.Ctx
}

func (c *Context) JSON(data interface{}, status ...int) error {
	if len(status) == 0 {
		status = append(status, fiber.StatusOK)
	}
	return c.Ctx.Status(status[0]).JSON(data)
}
