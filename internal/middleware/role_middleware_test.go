package middleware

import (
		"net/http/httptest"
	"testing"

	"github.com/365Soft-Bolivia/inmobiliaria-backend/internal/models"
	"github.com/gofiber/fiber/v2"
)

// Pruebas unitarias de AdminOrAdvisorMiddleware (internal/middleware/auth.go):
// verificaciÃ³n de rol para el acceso al panel de administraciÃ³n.
// La rama de agente/inmobiliaria consulta el perfil de asesor en la base de datos
// y se valida mediante la suite de integraciÃ³n (ver matriz de aceptaciÃ³n I5-03 e I5-04).

func appConMiddleware(handler func(c *fiber.Ctx) error) *fiber.App {
	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		isAdmin, _ := c.Locals("isAdmin").(bool)
		return c.JSON(fiber.Map{"ok": true, "isAdmin": isAdmin})
	})
	return app
}

func TestMiddlewareSinUsuarioRetorna401(t *testing.T) {
	app := appConMiddleware(AdminOrAdvisorMiddleware())
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("error en la peticiÃ³n: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("sin usuario autenticado debe retornar 401, obtuvo %d", resp.StatusCode)
	}
}

func TestMiddlewareRolUsuarioRetorna403(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &models.User{ID: 1, Rol: models.RoleUsuario})
		return c.Next()
	})
	app.Use(AdminOrAdvisorMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	if err != nil {
		t.Fatalf("error en la peticiÃ³n: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("rol usuario debe retornar 403, obtuvo %d", resp.StatusCode)
	}
}

func TestMiddlewareRolAdminPermiteAcceso(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &models.User{ID: 2, Rol: models.RoleAdmin})
		return c.Next()
	})
	app.Use(AdminOrAdvisorMiddleware())
	app.Get("/test", func(c *fiber.Ctx) error {
		isAdmin, _ := c.Locals("isAdmin").(bool)
		if !isAdmin {
			return c.Status(fiber.StatusInternalServerError).SendString("isAdmin no inyectado")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	if err != nil {
		t.Fatalf("error en la peticiÃ³n: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("rol admin debe retornar 200, obtuvo %d", resp.StatusCode)
	}
}

func TestMiddlewarePreflightOPTIONSPermitido(t *testing.T) {
	app := fiber.New()
	app.Use(AdminOrAdvisorMiddleware())
	// All permite comprobar que el middleware deja pasar OPTIONS sin usuario
	app.All("/test", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("error en la peticiÃ³n: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("preflight OPTIONS debe pasar sin verificaciÃ³n, obtuvo %d", resp.StatusCode)
	}
}
