package handlers

import (
	"../models"
	"github.com/gin-gonic/gin"
)

func BodyParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		body := models.Body{}
		c.Bind(&body)
		c.Set("user", body.User)
		c.Set("token", body.Token)
		c.Next()
	}
}
