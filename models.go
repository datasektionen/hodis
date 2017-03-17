package main

type Body struct {
	User
	Token
}

type User struct {
	Cn      string `form:"cn"      json:"cn"`
	Year    string `form:"year"    json:"year"`
	Uid     string `form:"uid"     json:"uid"`
	UgKthid string `form:"ugKthid" json:"ugKthid" gorm:"primary_key"`
	Refs    int
}

type Token struct {
	Value string `form:"token" json:"token"`
}

type Users []User

func (slice Users) Len() int {
	return len(slice)
}

func (slice Users) Less(i, j  int) bool {
	return slice[i].Refs > slice[j].Refs
}

func (slice Users) Swap(i, j int) {
    slice[i], slice[j] = slice[j], slice[i]
}
