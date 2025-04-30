package core

import (
	"net/http"
)

type Context struct {
	Request    *http.Request
	Writer     http.ResponseWriter
	Controller string
	Action     string
}

func (c *Context) Reset() {
	c.Request = nil
	c.Writer = nil
	c.Controller = ""
	c.Action = ""
}

func (c *Context) JSON(status int, data interface{}) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(status)
	_, err := c.Writer.Write([]byte(`{"message": "Handled by ` + c.Controller + `#` + c.Action + `"}`))
	return err
}

func (c *Context) String(status int, text string) error {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Writer.WriteHeader(status)
	_, err := c.Writer.Write([]byte(text))
	return err
}

func (c *Context) Param(name string) string {
	return ""
}

func (c *Context) Query(name string) string {
	return c.Request.URL.Query().Get(name)
}
