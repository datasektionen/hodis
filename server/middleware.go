package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/datasektionen/hodis/server/models"
	"github.com/datasektionen/hodis/server/pls"
	"github.com/gin-gonic/gin"
)

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Next()
	}
}

func bodyParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		body := models.Body{}
		c.Bind(&body)
		c.Set("data", body.User)
		c.Set("token", body.Token)
		c.Next()
	}
}

type verified struct {
	User string `json:"user"`
}

func (s *Server) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" ||
			c.Request.Method == "HEAD" ||
			c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		token := c.MustGet("token").(models.Token)
		if token.Login != "" {
			url := fmt.Sprintf("https://login.datasektionen.se/verify/%s?api_key=%s", token.Login, s.LoginKey)
			resp, err := http.Get(url)
			if err != nil {
				c.JSON(500, gin.H{"error": err})
				c.Abort()
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				c.JSON(401, gin.H{"error": "Access denied"})
				c.Abort()
				return
			}
			var res verified
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				c.JSON(500, gin.H{"error": "Failed to unmarshal json"})
				c.Abort()
				return
			}
			c.Set("uid", res.User)
			c.Set("pls", pls.HasPermission("user", res.User, "admin"))
			c.Next()
		} else if token.API != "" {
			c.Set("uid", "")
			c.Set("pls", pls.HasPermission("token", token.API, "admin"))
			c.Next()
		} else {
			c.JSON(400, gin.H{"error": "Missing token"})
			c.Abort()
		}
	}
}
