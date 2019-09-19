package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jinzhu/gorm"

	"github.com/gin-gonic/gin"
)

func Cache(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users Users
		db.Find(&users)
		c.JSON(200, gin.H{"cache": users, "length": len(users)})
	}
}

func UserSearch(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Param("query")

		res, err := Search(query)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			c.Abort()
		} else {
			c.JSON(200, res)
		}

	}
}

func Uid(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ExactUid(c.Param("uid"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			c.Abort()
		} else {
			c.JSON(200, res)
		}
	}
}

func UgKthid(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ExactUgid(c.Param("ugid"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			c.Abort()
		} else {
			c.JSON(200, res)
		}

	}
}

func Tag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user User
		db.Where(User{Tag: c.Param("tag")}).First(&user)
		if user.Uid != "" {
			c.JSON(200, user)
		} else {
			c.JSON(404, gin.H{"error": "Found no such tag"})
			c.Abort()
		}

	}
}

func Update(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.MustGet("data").(User)
		// Nobody can change UgKthid
		data.UgKthid = ""

		uid := c.MustGet("uid").(string)
		pls := c.MustGet("pls").(bool)

		if !pls {
			// No admin? No change
			data.Uid = ""
		}

		ExactUid(c.Param("uid"))
		if uid == c.Param("uid") || pls {
			var user User
			db.Where(User{Uid: c.Param("uid")}).Assign(data).FirstOrCreate(&user)
			c.JSON(200, user)
		} else {
			c.JSON(401, gin.H{"error": "Permission denied."})
		}

	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Next()
	}
}

func BodyParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		body := Body{}
		c.Bind(&body)
		c.Set("data", body.User)
		c.Set("token", body.Token)
		c.Next()
	}
}

func findToken(c *gin.Context) (string, bool) {
	if token := c.Query("token"); token != "" {
		return token, true
	} else if token := c.Query("api_key"); token != "" {
		return token, false
	}

	if token := c.PostForm("token"); token != "" {
		return token, true
	} else if token := c.PostForm("api_key"); token != "" {
		return token, false
	}

	token := c.MustGet("token").(Token)
	if token.Login != "" {
		return token.Login, true
	} else if token.API != "" {
		return token.API, false
	}

	return "", false
}

type Verified struct {
	User, FName, LName, Email, UgKthid string
}

func Authenticate(api_key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" ||
			c.Request.Method == "HEAD" ||
			c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		token := c.MustGet("token").(Token)
		if token.Login != "" {
			url := fmt.Sprintf("https://login.datasektionen.se/verify/%s?api_key=%s", token.Login, api_key)
			resp, err := http.Get(url)

			if err != nil {
				c.JSON(500, gin.H{"error": err})
				c.Abort()
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != 200 {
					c.JSON(401, gin.H{"error": "Access denied"})
					c.Abort()
				} else {
					var res Verified
					if json.NewDecoder(resp.Body).Decode(&res) == nil {
						c.Set("uid", res.User)
					}
					c.Set("pls", HasPlsPermission("user", res.User, "admin"))
					c.Next()
				}
			}
		} else if token.API != "" {
			c.Set("uid", "")
			c.Set("pls", HasPlsPermission("token", token.API, "admin"))
			c.Next()
		} else {
			c.JSON(400, gin.H{"error": "Missing token"})
			c.Abort()
		}
	}
}

func HasPlsPermission(token_type string, token_value string, permission string) bool {
	url := fmt.Sprintf("https://pls.datasektionen.se/api/%s/%s/hodis/%s", token_type, token_value, permission)
	resp, err := http.Get(url)
	if err != nil {
		return false
	} else {
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		if scanner.Scan() && scanner.Text() != "true" {
			return false
		}
	}
	return true
}

func Ping(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, "Pong")
	}
}
