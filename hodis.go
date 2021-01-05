package main

import (
	"log"
	"os"

	"github.com/datasektionen/hodis/handlers"
	"github.com/datasektionen/hodis/ldap"
	"github.com/datasektionen/hodis/models"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func main() {
	r := gin.Default()

	var db *gorm.DB
	var err error

	if gin.Mode() == gin.ReleaseMode {
		db, err = gorm.Open("postgres", os.Getenv("DATABASE_URL"))
		ldap.LdapInit("ldap.kth.se", 389, "ou=Addressbook,dc=kth,dc=se", db)
	} else {
		db, err = gorm.Open("sqlite3", "users.db")
		ldap.LdapInit("localhost", 9999, "ou=Addressbook,dc=kth,dc=se", db)
		r.GET("/cache", handlers.Cache(db))
	}
	if err != nil {
		log.Fatalln("Failed to connect database")
	}
	defer db.Close()
	db.AutoMigrate(&models.User{})

	r.Use(handlers.BodyParser())
	r.Use(handlers.CORS())

	loginKey := os.Getenv("LOGIN_API_KEY")
	if loginKey == "" {
		log.Fatalln("Please specify LOGIN_API_KEY")
	}
	r.Use(handlers.Authenticate(loginKey))

	r.GET("/users/:query", handlers.UserSearch(db))
	r.GET("/uid/:uid", handlers.Uid(db))
	r.GET("/ugkthid/:ugid", handlers.UgKthid(db))
	r.GET("/tag/:tag", handlers.Tag(db))
	r.GET("/ping", handlers.Ping(db))

	r.POST("/uid/:uid", handlers.Update(db))

	r.OPTIONS("/users/:query", func(c *gin.Context) {
		c.JSON(200, gin.H{})
	})
	r.OPTIONS("/uid/:uid", handlers.Uid(db))
	r.OPTIONS("/ugkthid/:ugid", handlers.UgKthid(db))

	r.Run()
}
