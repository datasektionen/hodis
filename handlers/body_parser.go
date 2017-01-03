package handlers

import (
	"../models"
	"github.com/gin-gonic/gin"
)

func BodyParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		body := models.Body{}
		c.Bind(&body)
		c.Set("body", body)
		c.Next()
	}
}
