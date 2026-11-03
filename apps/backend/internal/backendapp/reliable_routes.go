package backendapp

import (
	"github.com/gin-gonic/gin"
	"github.com/kandev/kandev/internal/reliable"
)

func registerReliableRoutes(router *gin.Engine, home string, addCleanup func(func() error)) error {
	handler, err := reliable.FromHome(home)
	if err != nil || handler == nil {
		return err
	}
	addCleanup(handler.Engine.Close)
	router.GET("/reliable", gin.WrapH(handler))
	router.GET("/reliable/", gin.WrapH(handler))
	router.GET("/reliable/icon.png", gin.WrapH(handler))
	router.GET("/reliable/background.png", gin.WrapH(handler))
	router.Any("/api/v1/reliable/*path", gin.WrapH(handler))
	return nil
}
