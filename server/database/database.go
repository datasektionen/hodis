package database

import (
	"github.com/datasektionen/hodis/server/models"

	"github.com/jinzhu/gorm"
)

type Database interface {
	AllUsers() []models.User
	UserFromTag(tag string) models.User
	Search(term string) []models.User
	ExactUID(uid string) models.User
	ExactUgKthID(ugkthid string) models.User
	AddUsers(users ...models.User)
	CreateOrUpdate(uid string, newUser models.User) models.User
}

type database struct {
	db *gorm.DB
}

func New(db *gorm.DB) Database {
	return &database{db: db}
}

func (d *database) AllUsers() []models.User {
	var users []models.User
	d.db.Find(&users)
	return users
}

func (d *database) UserFromTag(tag string) models.User {
	var user models.User
	d.db.Where(models.User{Tag: tag}).First(&user)
	return user
}

func (d *database) Search(query string) []models.User {
	var users []models.User
	d.db.Where("uid = ? OR ug_kthid = ? OR LOWER(cn) LIKE ?",
		query, query, "%"+query+"%").Find(&users)
	return users
}

func (d *database) ExactUID(uid string) models.User {
	var user models.User
	d.db.Where(models.User{Uid: uid}).First(&user)
	return user
}

func (d *database) ExactUgKthID(ugkthid string) models.User {
	var user models.User
	d.db.Where(models.User{UgKthid: ugkthid}).First(&user)
	return user
}

func (d *database) AddUsers(users ...models.User) {
	for _, u := range users {
		var user models.User
		d.db.First(&user, "ug_kthid = ?", u.UgKthid)
		if user.Uid == "" {
			d.db.Create(&u)
		} else {
			// Update names to match new base DN
			// These should be removed in the future
			user.Cn = u.Cn
			user.GivenName = u.GivenName
			user.DisplayName = u.DisplayName
			user.Mail = u.Mail

			d.db.Save(&user)
		}
	}
}

func (d *database) CreateOrUpdate(uid string, newUser models.User) models.User {
	var user models.User
	d.db.Where(models.User{Uid: uid}).Assign(newUser).FirstOrCreate(&user)
	return user
}
