package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Verified struct {
	User, FName, LName, Email, UgKthid string
}

type Token struct {
	Value string `form:"token" json:"token" required:true`
}

func findToken(c *gin.Context) (string, bool) {
	if token := c.Query("token"); token != "" {
		return token, true
	}

	if token := c.PostForm("token"); token != "" {
		return token, true
	}

	token := c.MustGet("body").(Body).Token
	if token.Value != "" {
		return token.Value, true
	}

	return "", false
}

func DAuth(api_key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" {
			c.Next()
			return
		}

		token, ok := findToken(c)
		if !ok {
			c.JSON(400, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		url := fmt.Sprintf("https://login2.datasektionen.se/verify/%s.json?api_key=%s", token, api_key)
		resp, err := http.Get(url)

		if err != nil {
			c.JSON(500, gin.H{"error": err})
			c.Abort()
		} else {
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				fmt.Println(resp)
				fmt.Println(token)
				c.JSON(401, gin.H{"error": "Access denied"})
				c.Abort()
			} else {
				var res Verified
				if json.NewDecoder(resp.Body).Decode(&res) == nil {
					c.Set("uid", res.User)
				}
				c.Next()
			}
		}
	}
}
