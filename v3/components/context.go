package components

import (
	"errors"
	"net/http"

	"github.com/go-raptor/errs"
	"github.com/labstack/echo/v4"
)

type Context struct {
	echo.Context
	Controller string
	Action     string
}

func (c *Context) JSON(data interface{}, status ...int) error {
	if len(status) == 0 {
		status = append(status, http.StatusOK)
	}
	return c.Context.JSON(status[0], data)
}

func (c *Context) JSONError(err error, status ...int) error {
	var e *errs.Error
	if errors.As(err, &e) {
		c.JSON(e, e.Code)
		return nil
	}

	if len(status) == 0 {
		status = append(status, http.StatusInternalServerError)
	}
	c.JSON(errs.NewError(status[0], err.Error()), status[0])
	return nil
}
