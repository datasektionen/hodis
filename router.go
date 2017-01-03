package main

import (
	"os"

	"./ldap"

	"github.com/gin-gonic/gin"

    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/sqlite"
)

type User struct {
	Cn      string `form:"cn"      json:"cn"`
	Year    string `form:"year"    json:"year"`
	Uid     string `form:"uid"     json:"uid"`
	UgKthid string `form:"ugKthid" json:"ugKthid" gorm:"primary_key"`
}

func main() {
	r := gin.Default()

	if gin.Mode() == gin.ReleaseMode {
		ldap.Init("ldap.kth.se", 389, "ou=unix,dc=kth,dc=se")
	} else {
		ldap.Init("localhost", 9999, "ou=unix,dc=kth,dc=se")
	}
	
	db, err := gorm.Open("sqlite3", "users.db")
    if err != nil {
        panic("failed to connect database")
    }
    defer db.Close()

    db.AutoMigrate(&User{})

    r.Use(BodyParser())
	r.Use(AccessControl())

	login_key := os.Getenv("LOGIN_API_KEY")
	if login_key != "" {
		r.Use(DAuth(login_key))
	}

	r.GET("/cache", Cache(db))
	r.GET("/users/:query", UserSearch(db))
	r.GET("/uid/:uid", Uid(db))	
	r.GET("/ugkthid/:ugid", UgKthid(db))

	r.POST("/uid/:uid", Update(db))

	r.Run()
}

type Body struct {
	User
	Token
}

func BodyParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		body := Body{}
		c.Bind(&body)
		c.Set("body", body)
		c.Next()
	}
}

func AccessControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")

		c.Next()
	}
}
