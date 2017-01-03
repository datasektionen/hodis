package models

type Body struct {
	User
	Token
}

type User struct {
	Cn      string `form:"cn"      json:"cn"`
	Year    string `form:"year"    json:"year"`
	Uid     string `form:"uid"     json:"uid"`
	UgKthid string `form:"ugKthid" json:"ugKthid" gorm:"primary_key"`
}

type Token struct {
	Value string `form:"token" json:"token"`
}
