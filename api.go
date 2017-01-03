package main

import (
	"./ldap"
	
	"github.com/gin-gonic/gin"
    
    "github.com/jinzhu/gorm"
)

func Cache(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []User
		db.Find(&users)
		c.JSON(200, gin.H{"cache": users, "length": len(users)})
	}
}

func UserSearch(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Param("query")

		res, err := ldap.SearchWithCache(ldap.User{Uid: query, Cn: query, UgKthid: query}, false)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			c.Abort()
		}

		for _, ldapUser := range res {
			var users []User
			db.Where(User{Uid: ldapUser.Uid}).Find(&users)

			if len(users) == 0 {
				user := User{
					Cn: ldapUser.Cn,
					Uid: ldapUser.Uid,
					UgKthid: ldapUser.UgKthid,
				}
				db.Create(&user)
			}
		}

		c.JSON(200, res)
	}
}

func Uid(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ldap.SearchWithCache(ldap.User{Uid: c.Param("uid")}, true)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			c.Abort()
		}

		if len(res) != 1 {
			c.JSON(404, gin.H{"error": "User not found"})
		}

		c.JSON(200, res[0])
	}
}

func Update(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.MustGet("body").(Body).User
		uid := c.MustGet("uid").(string)
		if uid != c.Param("uid") {
			c.JSON(401, gin.H{"error": "Permission denied."})
			c.Abort()
			return
		}

		var user User
		db.Where(User{Uid: uid}).Assign(data).FirstOrCreate(&user)

		c.JSON(200, user)
	}
}

func UgKthid(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ldap.SearchWithCache(ldap.User{UgKthid: c.Param("ugid")}, true)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			c.Abort()
		}

		if len(res) != 1 {
			c.JSON(404, gin.H{"error": "User not found"})
		}

		c.JSON(200, res[0])
	}
}
