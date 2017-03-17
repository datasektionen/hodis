package handlers

import (
	"../ldap"
	"../models"
	
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
		query := c.Param("query")

		res, err := ldap.Search(query)
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
		res, err := ldap.ExactUid(c.Param("uid"))
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
		data := c.MustGet("user").(models.User)
		uid := c.MustGet("uid").(string)
		if uid == c.Param("uid") || HasPlsPermission(uid, "hodis", "admin") {
			var user models.User
			db.Where(models.User{Uid: c.Param("uid")}).Assign(data).FirstOrCreate(&user)
			c.JSON(200, user)
		} else {
			c.JSON(401, gin.H{"error": "Permission denied."})
		}

	}
}

func UgKthid(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ldap.ExactUgid(c.Param("ugid"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			c.Abort()
		} else {
			c.JSON(200, res)
		}

	}
}
