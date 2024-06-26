package raptor

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

type Map map[string]interface{}

type Context struct {
	*fiber.DefaultCtx
	Controller string
	Action     string
}

func (c *Context) JSON(data interface{}, status ...int) error {
	if len(status) == 0 {
		status = append(status, fiber.StatusOK)
	}
	return c.DefaultCtx.Status(status[0]).JSON(data)
}

func (c *Context) JSONError(err error, status ...int) error {
	var e *Error
	if errors.As(err, &e) {
		return c.DefaultCtx.Status(e.Code).JSON(e)
	}

	if len(status) == 0 {
		status = append(status, http.StatusInternalServerError)
	}
	return c.DefaultCtx.Status(status[0]).JSON(NewError(status[0], err.Error()))
}
