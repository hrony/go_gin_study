package middleware

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"runtime/debug"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(string(debug.Stack()))

				public.ComLogNotice(c, "_com_panic", map[string]interface{}{
					"error": fmt.Sprint(err),
					"stack": string(debug.Stack()),
				})

				if lib.ConfBase.DebugMode != "debug" {
					ResponseError(c, 500, errors.New("内部错误"))
					return
				}
				ResponseError(c, 500, errors.New(fmt.Sprint(err)))
				return
			}
		}()
		c.Next()
	}
}
