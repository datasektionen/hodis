package models

import "time"

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

	// The date they are currently known to be chapter members until. `MemberToForever` is a special
	// case which means that the person should never stop being considered a chapter member. This
	// should overrule dates given by THS. TODO: change this before year 9998.
	MemberTo *time.Time `form:"memberTo" json:"memberTo"`
	Year     int        `form:"year"     json:"year"`
	Tag      string     `form:"tag"      json:"tag"`

	Refs uint `json:"-"`
}

var MemberToForever time.Time

func init() {
	var err error
	MemberToForever, err = time.Parse(time.DateOnly, "9999-12-24")
	if err != nil {
		panic(err)
	}
}

type Token struct {
	Login string `form:"token" json:"token"`
	API   string `form:"api_key" json:"api_key"`
}

type Users []User
