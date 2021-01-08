package main

import (
	"log"
	"os"

	"github.com/datasektionen/hodis/server"
	"github.com/datasektionen/hodis/server/database"
	"github.com/datasektionen/hodis/server/ldap"
	"github.com/datasektionen/hodis/server/models"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func main() {
	var (
		gormDB *gorm.DB
		l      ldap.Ldap
		err    error
	)
	if gin.Mode() == gin.ReleaseMode {
		l = ldap.New("ldap.kth.se:389")
		gormDB, err = gorm.Open("postgres", os.Getenv("DATABASE_URL"))
	} else {
		l = ldap.New("localhost:9999")
		gormDB, err = gorm.Open("sqlite3", "users.db")
	}
	if err != nil {
		log.Fatalln("Failed to connect database")
	}
	defer gormDB.Close()
	gormDB.AutoMigrate(&models.User{})

	db := database.New(gormDB)

	loginKey := os.Getenv("LOGIN_API_KEY")
	if loginKey == "" {
		log.Fatalln("Please specify LOGIN_API_KEY")
	}

	s := &server.Server{DB: db, LoginKey: loginKey, Ldap: l}
	s.Router().Run()
}
