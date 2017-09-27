package main

type Body struct {
	User
	Token
}

type User struct {
	Cn      string `form:"cn"      json:"cn"`
	Year    int    `form:"year"    json:"year"`
	Uid     string `form:"uid"     json:"uid"`
	UgKthid string `form:"ugKthid" json:"ugKthid" gorm:"primary_key"`
	Refs    uint   `json:"-"`
}

type Token struct {
	Login string `form:"token" json:"token"`
	API   string `form:"api_key" json:"api_key"`
}

type Users []User
