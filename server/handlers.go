package server

import (
	"log"
	"strings"

	"github.com/datasektionen/hodis/server/models"
	"github.com/gin-gonic/gin"
)

func (s *Server) cache() gin.HandlerFunc {
	return func(c *gin.Context) {
		users := s.DB.AllUsers()
		c.JSON(200, gin.H{"cache": users, "length": len(users)})
	}
}

func (s *Server) userSearch() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := strings.ToLower(c.Param("query"))
		dbRes := s.DB.Search(query)
		if _, ok := s.queries.Load(query); ok || len(dbRes) >= 1000 {
			c.JSON(200, dbRes)
			return
		}

		ldapRes, err := s.Ldap.Search(query)
		if err != nil {
			log.Printf("userSearch() failed: %v", err)
			c.JSON(500, gin.H{"error": "search failed"})
			return
		}

		s.DB.AddUsers(ldapRes...)
		s.queries.Store(query, nil)
		c.JSON(200, uniqueUsers(ldapRes, dbRes))
	}
}

func (s *Server) uid() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.Param("uid")
		if user := s.DB.ExactUID(uid); user.UgKthid != "" {
			c.JSON(200, user)
			return
		}

		user, err := s.Ldap.ExactUid(uid)
		if err != nil {
			log.Printf("uid() failed: %v", err)
			c.JSON(404, gin.H{"error": "user not found"})
			return
		}
		s.DB.AddUsers(user)
		c.JSON(200, user)
	}
}

func (s *Server) ugKthid() gin.HandlerFunc {
	return func(c *gin.Context) {
		ugkthid := c.Param("ugid")
		if user := s.DB.ExactUgKthID(ugkthid); user.UgKthid != "" {
			c.JSON(200, user)
			return
		}

		user, err := s.Ldap.ExactUgid(ugkthid)
		if err != nil {
			log.Printf("ugkthid() failed: %v", err)
			c.JSON(404, gin.H{"error": "user not found"})
			return
		}
		s.DB.AddUsers(user)
		c.JSON(200, user)
	}
}

func (s *Server) tag() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := s.DB.UserFromTag(c.Param("tag"))
		if user.UgKthid == "" {
			c.JSON(404, gin.H{"error": "Found no such tag"})
			return
		}
		c.JSON(200, user)
	}
}

func (s *Server) update() gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.MustGet("data").(models.User)
		// Nobody can change UgKthid
		data.UgKthid = ""

		uid := c.MustGet("uid").(string)
		pls := c.MustGet("pls").(bool)
		if !pls {
			// No admin? No change
			data.Uid = ""
		}

		s.Ldap.ExactUid(c.Param("uid"))
		if uid != c.Param("uid") && !pls {
			c.JSON(401, gin.H{"error": "Permission denied."})
			return
		}
		user := s.DB.CreateOrUpdate(c.Param("uid"), data)
		c.JSON(200, user)
	}
}

func ping() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, "Pong")
	}
}
