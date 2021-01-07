package server

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

func Setup(db *gorm.DB, loginKey string) *gin.Engine {
	r := gin.Default()

	if gin.Mode() != gin.ReleaseMode {
		r.GET("/cache", cache(db))
	}

	r.Use(bodyParser())
	r.Use(cors())

	r.Use(authenticate(loginKey))

	r.GET("/users/:query", userSearch(db))
	r.GET("/uid/:uid", uid(db))
	r.GET("/ugkthid/:ugid", ugKthid(db))
	r.GET("/tag/:tag", tag(db))
	r.GET("/ping", ping(db))

	r.POST("/uid/:uid", update(db))

	r.OPTIONS("/users/:query", func(c *gin.Context) {
		c.JSON(200, gin.H{})
	})
	r.OPTIONS("/uid/:uid", uid(db))
	r.OPTIONS("/ugkthid/:ugid", ugKthid(db))

	return r
}
