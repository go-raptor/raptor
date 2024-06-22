package raptor

import "github.com/gofiber/fiber/v3"

type Template struct {
	Engine fiber.Views
	Layout string
}
