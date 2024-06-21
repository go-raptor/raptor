package raptor

import "github.com/gofiber/fiber/v2"

type Template struct {
	Engine fiber.Views
	Layout string
}
