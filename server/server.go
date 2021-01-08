package server

import (
	"sync"

	"github.com/datasektionen/hodis/server/database"
	"github.com/datasektionen/hodis/server/ldap"

	"github.com/gin-gonic/gin"
)

type Server struct {
	queries  sync.Map
	DB       database.Database
	LoginKey string
	Ldap     ldap.Ldap
}

func (s *Server) Router() *gin.Engine {
	r := gin.Default()

	if gin.Mode() != gin.ReleaseMode {
		r.GET("/cache", s.cache())
	}

	r.Use(bodyParser())
	r.Use(cors())

	r.Use(s.authenticate())

	r.GET("/users/:query", s.userSearch())
	r.GET("/uid/:uid", s.uid())
	r.GET("/ugkthid/:ugid", s.ugKthid())
	r.GET("/tag/:tag", s.tag())
	r.GET("/ping", ping())

	r.POST("/uid/:uid", s.update())

	r.OPTIONS("/users/:query", func(c *gin.Context) {
		c.JSON(200, gin.H{})
	})
	r.OPTIONS("/uid/:uid", s.uid())
	r.OPTIONS("/ugkthid/:ugid", s.ugKthid())

	return r
}
