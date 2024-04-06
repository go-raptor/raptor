package raptor

import (
	"net"

	"github.com/gofiber/fiber/v2"
)

type Map map[string]interface{}

type Context struct {
	*fiber.Ctx
	Controller string
	Action     string
}

func (c *Context) JSON(data interface{}, status ...int) error {
	if len(status) == 0 {
		status = append(status, fiber.StatusOK)
	}
	return c.Ctx.Status(status[0]).JSON(data)
}

func (c *Context) PublicIP() string {
	if c.Ctx.IPs() != nil && len(c.Ctx.IPs()) > 0 {
		for _, ip := range c.Ctx.IPs() {
			parsedIP := net.ParseIP(ip)
			if !parsedIP.IsPrivate() {
				return ip
			}
		}
	}

	return c.Ctx.IP()
}
