package main

import (
	"log"
	"os"
	"strconv"

	"github.com/datasektionen/hodis/handlers"
	"github.com/datasektionen/hodis/ldap"
	"github.com/datasektionen/hodis/models"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func main() {
	r := gin.Default()

	var db *gorm.DB
	var err error

	db, err = gorm.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalln("Failed to connect to database", err)
	}

	ldapHost := os.Getenv("LDAP_HOST")
	if ldapHost == "" {
		ldapHost = "ldap.kth.se"
	}
	ldapPort := 389
	if ldapPortStr := os.Getenv("LDAP_PORT"); ldapPortStr != "" {
		ldapPort, err = strconv.Atoi(ldapPortStr)
		if err != nil {
			log.Fatalln("Invalid number in $LDAP_PORT", err)
		}
	}
	ldap.LdapInit(ldapHost, ldapPort, "ou=Addressbook,dc=kth,dc=se", db)

	defer db.Close()
	db.AutoMigrate(&models.User{})

	if gin.Mode() != gin.ReleaseMode {
		r.GET("/cache", handlers.Cache(db))
	}

	loginURL := os.Getenv("LOGIN_URL")
	if loginURL == "" {
		loginURL = "https://login.datasektionen.se"
	}
	loginKey := os.Getenv("LOGIN_API_KEY")
	if loginKey == "" {
		log.Fatalln("Please specify LOGIN_API_KEY")
	}
	plsURL := os.Getenv("PLS_URL")
	if loginKey == "" {
		plsURL = "https://pls.datasektionen.se"
	}
	auth := handlers.Authenticate(loginURL, loginKey, plsURL)

	r.POST("/membership-sheet", handlers.HeaderParser, auth, handlers.MembershipSheet(db))

	r.Use(handlers.BodyParser())
	r.Use(handlers.CORS())

	r.Use(auth)

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
