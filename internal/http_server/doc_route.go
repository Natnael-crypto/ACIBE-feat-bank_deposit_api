package httpserver

import (
	_ "net/http/pprof"

	_ "github.com/aci/backend/docs"
	"github.com/gin-contrib/pprof"
	ginSwagger "github.com/swaggo/gin-swagger"

	swaggerFiles "github.com/swaggo/files"
)

func (a *App) swaggerRoute() {
	pprof.Register(a.gin)
	a.gin.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
