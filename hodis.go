package main

import (
	"os"

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
		LdapInit("ldap.kth.se", 389, "ou=unix,dc=kth,dc=se", db)
	} else {
		db, err = gorm.Open("sqlite3", "users.db")
		LdapInit("localhost", 9999, "ou=unix,dc=kth,dc=se", db)
	}

	if err != nil {
		panic("Failed to connect database")
	}
	defer db.Close()
	db.AutoMigrate(&User{})

	r.Use(BodyParser())
	r.Use(CORS())

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
