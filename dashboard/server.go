package dashboard

import (
	"context"
	"gott"

	"go.uber.org/zap"

	"github.com/labstack/echo"
)

var HTTP *echo.Echo

func Serve(b *gott.Broker, address string) {
	HTTP = echo.New()
	handle(b)
	gott.GOTT.Logger.Fatal("dashboard web server error", zap.Error(HTTP.Start(address)))
}

func Stop() {
	if HTTP != nil {
		_ = HTTP.Shutdown(context.Background())
	}
}
