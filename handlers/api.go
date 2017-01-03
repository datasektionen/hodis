package handlers

import (
	"../ldap"
	"../models"
	
	"github.com/gin-gonic/gin"

	"github.com/jinzhu/gorm"

	"github.com/bradfitz/slice"
)

func Cache(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var users []models.User
		db.Find(&users)
		c.JSON(200, gin.H{"cache": users, "length": len(users)})
	}
}

func UserSearch(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Param("query")

		res, err := ldap.SearchWithDb(models.User{Uid: query, Cn: query, UgKthid: query}, false)
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
		res, err := ldap.SearchWithDb(models.User{Uid: c.Param("uid")}, true)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			c.Abort()
		}

		if len(res) != 1 {
			c.JSON(404, gin.H{"error": "User not found"})
		} else {
			c.JSON(200, res[0])
		}
	}
}

func Update(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.MustGet("body").(models.Body).User
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
		res, err := ldap.SearchWithDb(models.User{UgKthid: c.Param("ugid")}, true)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			c.Abort()
		} else if len(res) != 1 {
			c.JSON(404, gin.H{"error": "User not found"})
		} else {
			c.JSON(200, res[0])
		}

	}
}
