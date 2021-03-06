package middleware

import (
	"github.com/gin-gonic/gin"
	"go_gin_study/lesson/Gin入门实战/demo/public"
	en_translations "gopkg.in/go-playground/validator.v9/translations/en"
	zh_translations "gopkg.in/go-playground/validator.v9/translations/zh"
	zh_tw_translations "gopkg.in/go-playground/validator.v9/translations/zh_tw"
)

func TranslationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := c.DefaultQuery("locale", "zh")
		trans, _ := public.Uni.GetTranslator(locale)
		switch locale {
		case "zh":
			zh_translations.RegisterDefaultTranslations(public.Validate, trans)
			break
		case "en":
			en_translations.RegisterDefaultTranslations(public.Validate, trans)
			break
		case "zh_tw":
			zh_tw_translations.RegisterDefaultTranslations(public.Validate, trans)
			break
		default:
			zh_translations.RegisterDefaultTranslations(public.Validate, trans)
			break
		}
		c.Set("trans", trans)
		c.Next()
	}
}
