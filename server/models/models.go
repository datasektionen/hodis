package models

type Body struct {
	User
	Token
}

type User struct {
	UgKthid     string `form:"ugKthid"     json:"ugKthid" gorm:"primary_key"`
	Uid         string `form:"uid"         json:"uid"`
	Cn          string `form:"cn"          json:"cn"`
	Mail        string `form:"mail"        json:"mail"`
	GivenName   string `form:"givenName"   json:"givenName"`
	DisplayName string `form:"displayName" json:"displayName"`

	Year int    `form:"year"        json:"year"`
	Tag  string `form:"tag"         json:"tag"`

	Refs uint `json:"-"`
}

type Token struct {
	Login string `form:"token" json:"token"`
	API   string `form:"api_key" json:"api_key"`
}

type Users []User
