package gott

import (
	"context"

	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"

	"github.com/labstack/echo"
)

func newDashboardServer(b *Broker) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize: 1 << 10, // 1 KB
		LogLevel:  log.ERROR,
	}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "SAMEORIGIN",
		//HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self' 'unsafe-inline'",
	}))

	initDashboardRoutes(e, b)

	return e
}

func stopDashboardServer(server *echo.Echo) {
	if server != nil {
		_ = server.Shutdown(context.Background())
	}
}
