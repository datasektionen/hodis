package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/datasektionen/hodis/ldap"
	"github.com/datasektionen/hodis/models"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

func Cache(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users models.Users
		db.Find(&users)
		c.JSON(200, gin.H{"cache": users, "length": len(users)})
	}
}

func UserSearch(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ldap.Search(c.Param("query"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, res)
	}
}

func Uid(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ldap.ExactUid(c.Param("uid"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, res)
	}
}

func UgKthid(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ldap.ExactUgid(c.Param("ugid"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, res)
	}
}

func Tag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		db.Where(models.User{Tag: c.Param("tag")}).First(&user)
		if user.Uid == "" {
			c.JSON(404, gin.H{"error": "Found no such tag"})
			return
		}
		c.JSON(200, user)
	}
}

func Update(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.MustGet("data").(models.User)
		// Nobody can change UgKthid
		data.UgKthid = ""

		uid := c.MustGet("uid").(string)
		pls := c.MustGet("pls").(bool)
		if !pls {
			// No admin? No change
			data.Uid = ""
		}

		ldap.ExactUid(c.Param("uid"))
		if uid != c.Param("uid") && !pls {
			c.JSON(401, gin.H{"error": "Permission denied."})
			return
		}
		var user models.User
		db.Where(models.User{Uid: c.Param("uid")}).Assign(data).FirstOrCreate(&user)
		c.JSON(200, user)
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

func Authenticate(loginURL, apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" ||
			c.Request.Method == "HEAD" ||
			c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		token := c.MustGet("token").(models.Token)
		if token.Login != "" {
			url := fmt.Sprintf("%s/verify/%s?api_key=%s", loginURL, token.Login, apiKey)
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
			c.Set("pls", HasPlsPermission("user", res.User, "admin"))
			c.Next()
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

func HasPlsPermission(tokenType, tokenValue, permission string) bool {
	url := fmt.Sprintf("https://pls.datasektionen.se/api/%s/%s/hodis/%s", tokenType, tokenValue, permission)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if b, err := ioutil.ReadAll(resp.Body); err != nil || string(b) != "true" {
		return false
	}
	return true
}

func Ping(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, "Pong")
	}
}
