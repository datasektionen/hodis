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

func Update(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.MustGet("user").(User)
		uid := c.MustGet("uid").(string)
		ExactUid(c.Param("uid"))
		if uid == c.Param("uid") || HasPlsPermission(uid, "hodis", "admin") {
			var user User
			db.Where(User{Uid: c.Param("uid")}).Assign(data).FirstOrCreate(&user)
			c.JSON(200, user)
		} else {
			c.JSON(401, gin.H{"error": "Permission denied."})
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
		c.Set("user", body.User)
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

func DAuth(api_key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" ||
			c.Request.Method == "HEAD" ||
			c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		token, ok := findToken(c)
		if !ok {
			//was actually api key
			if token != "" {
				c.Set("uid", token)
				return
			} else {
				c.JSON(400, gin.H{"error": "Missing token"})
				c.Abort()
				return
			}
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

func HasPlsPermission(uid string, system string, permission string) bool {
	url := fmt.Sprintf("https://pls.datasektionen.se/api/user/%s/%s/%s", uid, system, permission)
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
