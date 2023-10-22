package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/datasektionen/hodis/ldap"
	"github.com/datasektionen/hodis/models"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/xuri/excelize/v2"
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
		res, err := ldap.Search(c.Param("query"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, res)
	}
}

func Uid(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ldap.ExactUid(c.Param("uid"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, res)
	}
}

func UgKthid(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := ldap.ExactUgid(c.Param("ugid"))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, res)
	}
}

func Tag(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.User
		db.Where(models.User{Tag: c.Param("tag")}).First(&user)
		if user.Uid == "" {
			c.JSON(404, gin.H{"error": "Found no such tag"})
			return
		}
		c.JSON(200, user)
	}
}

func Update(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := c.MustGet("data").(models.User)
		// Nobody can change UgKthid
		data.UgKthid = ""

		uid := c.MustGet("uid").(string)
		admin := c.MustGet("admin").(bool)
		if !admin {
			// No admin? No change
			data.Uid = ""
		}

		ldap.ExactUid(c.Param("uid"))
		if uid != c.Param("uid") && !admin {
			c.JSON(401, gin.H{"error": "Permission denied."})
			return
		}
		var user models.User
		db.Where(models.User{Uid: c.Param("uid")}).Assign(data).FirstOrCreate(&user)
		c.JSON(200, user)
	}
}

func MembershipSheet(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin := c.MustGet("admin").(bool)
		if !admin {
			c.JSON(401, gin.H{"error": "Permission denied."})
			return
		}

		sheet, err := excelize.OpenReader(c.Request.Body)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		if sheet.SheetCount < 1 {
			c.JSON(400, gin.H{"error": "found no sheets in the provided file"})
			c.Abort()
			return
		}
		rows, err := sheet.Rows(sheet.GetSheetName(0))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		if !rows.Next() {
			c.JSON(400, gin.H{"error": "no header row found"})
			c.Abort()
			return
		}
		columns, err := rows.Columns()
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		var dateCol, emailCol, chapterCol int = -1, -1, -1
		for i, title := range columns {
			title = strings.TrimSpace(title)
			if title == "Giltig till" {
				dateCol = i
			} else if title == "E-postadress" {
				emailCol = i
			} else if title == "Grupp" {
				chapterCol = i
			}
		}
		if dateCol == -1 {
			c.JSON(400, gin.H{"error": "couldn't find a column for dates"})
			c.Abort()
			return
		}
		if emailCol == -1 {
			c.JSON(400, gin.H{"error": "couldn't find a column for emails"})
			c.Abort()
			return
		}
		if chapterCol == -1 {
			c.JSON(400, gin.H{"error": "couldn't find a column for chapters"})
			c.Abort()
			return
		}
		now := time.Now()
		errorRows := make([]string, 0)
		for rows.Next() {
			columns, err := rows.Columns()
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				c.Abort()
				return
			}
			if len(columns) == 0 {
				continue
			}
			if dateCol >= len(columns) || emailCol >= len(columns) || chapterCol >= len(columns) {
				log.Printf("Some column (of %d, %d, %d) not found on row with length %d\n", dateCol, emailCol, chapterCol, len(columns))
				errorRows = append(errorRows, strings.Join(columns, ","))
				continue
			}
			date := columns[dateCol]
			email := columns[emailCol]
			chapter := columns[chapterCol]

			if !strings.Contains(chapter, "Datasektionen") {
				if err := db.Model(models.User{}).
					Where(models.User{Mail: email}).
					Updates(models.User{MemberTo: &now}).
					Error; err != nil {
					log.Println(email, err)
					errorRows = append(errorRows, email)
				}
				continue
			}

			memberTo, err := time.Parse(time.DateOnly, date)

			// Note that although very common, everyone's kth email is not their kth id followed by
			// @kth.se. KTH:s ldap can however not be searched by emails, so I don't know of a good
			// solution.
			uid := strings.TrimSuffix(email, "@kth.se")
			var user models.User
			db.Where(models.User{Uid: uid}).First(&user)
			inDB := user.UgKthid != ""
			user, err = ldap.ExactUid(uid)
			if err != nil {
				log.Println(email, err)
				errorRows = append(errorRows, email)
				continue
			}
			user.MemberTo = &memberTo
			err = nil
			if inDB {
				err = db.Save(&user).Error
			} else {
				err = db.Create(&user).Error
			}
			if err != nil {
				log.Println(email, err)
				errorRows = append(errorRows, email)
				continue
			}
		}
		c.JSON(200, gin.H{"erroring-rows": errorRows})
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Next()
	}
}

func BodyParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		body := models.Body{}
		c.Bind(&body)
		c.Set("data", body.User)
		c.Set("token", body.Token)
		c.Next()
	}
}

func HeaderParser(c *gin.Context) {
	c.Set("token", models.Token{
		Login: c.GetHeader("X-Token"),
		API:   c.GetHeader("X-Api-Key"),
	})
}

type verified struct {
	User string `json:"user"`
}

func Authenticate(loginURL, apiKey, plsURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" ||
			c.Request.Method == "HEAD" ||
			c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		token := c.MustGet("token").(models.Token)
		if token.Login != "" {
			url := fmt.Sprintf("%s/verify/%s?api_key=%s", loginURL, token.Login, apiKey)
			resp, err := http.Get(url)
			if err != nil {
				c.JSON(500, gin.H{"error": err})
				c.Abort()
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				c.JSON(401, gin.H{"error": "Access denied"})
				c.Abort()
				return
			}
			var res verified
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				c.JSON(500, gin.H{"error": "Failed to unmarshal json"})
				c.Abort()
				return
			}
			c.Set("uid", res.User)
			c.Set("admin", HasPlsPermission(plsURL, "user", res.User, "admin"))
			c.Next()
		} else if token.API != "" {
			c.Set("uid", "")
			c.Set("admin", HasPlsPermission(plsURL, "token", token.API, "admin"))
			c.Next()
		} else {
			c.JSON(400, gin.H{"error": "Missing token"})
			c.Abort()
		}
	}
}

func HasPlsPermission(plsURL, tokenType, tokenValue, permission string) bool {
	if plsURL == "true" {
		return true
	} else if plsURL == "false" {
		return false
	}
	url := fmt.Sprintf("%s/api/%s/%s/hodis/%s", plsURL, tokenType, tokenValue, permission)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if b, err := ioutil.ReadAll(resp.Body); err != nil || string(b) != "true" {
		return false
	}
	return true
}

func Ping(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, "Pong")
	}
}
