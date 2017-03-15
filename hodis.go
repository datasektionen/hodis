package main

import (
	"os"

	"./handlers"
	"./ldap"
	"./models"

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
		ldap.Init("ldap.kth.se", 389, "ou=unix,dc=kth,dc=se", db)
	} else {
		db, err = gorm.Open("sqlite3", "users.db")
		ldap.Init("localhost", 9999, "ou=unix,dc=kth,dc=se", db)
	}

	if err != nil {
		panic("Failed to connect database")
	}
	defer db.Close()
	db.AutoMigrate(&models.User{})

	r.Use(handlers.BodyParser())
	r.Use(handlers.CORS())

	login_key := os.Getenv("LOGIN_API_KEY")
	if login_key != "" {
		r.Use(handlers.DAuth(login_key))
	}

	r.GET("/cache", handlers.Cache(db))
	r.GET("/users/:query", handlers.UserSearch(db))
	r.GET("/uid/:uid", handlers.Uid(db))
	r.GET("/ugkthid/:ugid", handlers.UgKthid(db))

	r.POST("/uid/:uid", handlers.Update(db))

	r.Run()
}
