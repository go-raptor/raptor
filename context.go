package raptor

import "github.com/gofiber/fiber/v2"

type Context struct {
	*fiber.Ctx
}

type Map map[string]interface{}
