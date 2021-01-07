package main

import (
	"log"
	"os"

	"github.com/datasektionen/hodis/server"
	"github.com/datasektionen/hodis/server/ldap"
	"github.com/datasektionen/hodis/server/models"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func main() {
	var db *gorm.DB
	var err error
	if gin.Mode() == gin.ReleaseMode {
		db, err = gorm.Open("postgres", os.Getenv("DATABASE_URL"))
		ldap.LdapInit("ldap.kth.se", 389, "ou=Addressbook,dc=kth,dc=se", db)
	} else {
		db, err = gorm.Open("sqlite3", "users.db")
		ldap.LdapInit("localhost", 9999, "ou=Addressbook,dc=kth,dc=se", db)
	}
	if err != nil {
		log.Fatalln("Failed to connect database")
	}
	defer db.Close()
	db.AutoMigrate(&models.User{})

	loginKey := os.Getenv("LOGIN_API_KEY")
	if loginKey == "" {
		log.Fatalln("Please specify LOGIN_API_KEY")
	}

	r := server.Setup(db, loginKey)
	r.Run()
}
