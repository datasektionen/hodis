package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/jinzhu/gorm"
	"gopkg.in/ldap.v2"
)

type settings struct {
	queries map[string]uint

	ldap_host string
	ldap_port int

	base_dn    string
	attributes []string

	db *gorm.DB
}

var s settings

func LdapInit(ldap_host string, ldap_port int, base_dn string, db *gorm.DB) {
	s = settings{
		make(map[string]uint),
		ldap_host,
		ldap_port,
		base_dn,
		[]string{"cn", "uid", "ugKthid"},
		db,
	}
}

func Search(query string) (Users, error) {
	query = strings.ToLower(query)

	var db_results Users
	s.db.Where(User{Uid: query}).
		Or(User{UgKthid: query}).
		Or("LOWER(cn) LIKE ?", fmt.Sprintf("%%%s%%", query)).
		Find(&db_results)

	filter := fmt.Sprintf("(|(cn=*%s*)(uid=*%s*)(ugKthid=*%s*))", query, query, query)

	if len(db_results) >= 1000 || s.queries[query] > 0 {
		sort.Slice(db_results, func(i, j int) bool {
			return db_results[i].Refs > db_results[j].Refs
		})
		return db_results, nil
	}
	s.queries[query]++

	user_results, err := searchLDAP(filter)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %s", err.Error())
	}

	return uniqueUsers(user_results, db_results), nil
}

func ExactUid(query string) (User, error) {
	return exactSearch(query, "uid")
}

func ExactUgid(query string) (User, error) {
	return exactSearch(query, "ugKthid")
}

func exactSearch(query string, ldapField string) (User, error) {
	var user User
	userFound := false
	if ldapField == "uid" {
		s.db.Where(User{Uid: query}).First(&user)
		userFound = user.Uid != ""
	} else if ldapField == "ugKthid" {
		s.db.Where(User{UgKthid: query}).First(&user)
		userFound = user.UgKthid != ""
	}

	if userFound {
		user.Refs++
		s.db.Save(&user)
		return user, nil
	}

	users, err := searchLDAP(fmt.Sprintf("(%s=%s)", ldapField, query))

	if err != nil {
		return user, err
	}

	if len(users) == 1 {
		return users[0], nil
	} else if len(users) > 1 {
		return user, fmt.Errorf("Exact search failed: Got multiple results.")
	} else {
		return user, fmt.Errorf("Exact search failed: No such user exists.")
	}
}

func searchLDAP(filter string) ([]User, error) {
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", s.ldap_host, s.ldap_port))
	if err != nil {
		return nil, fmt.Errorf("Dial failed: %s", err.Error())
	}
	defer l.Close()

	ldapSearchRequest := ldap.NewSearchRequest(
		s.base_dn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		s.attributes,
		nil,
	)

	res, err := l.SearchWithPaging(ldapSearchRequest, 10)
	if err != nil {
		if len(res.Entries) > 0 {
			log.Printf("%s but %d results, continuing.", err.Error(), len(res.Entries))
		} else {
			return nil, fmt.Errorf("Paged search failed: %s", err.Error())
		}
	}

	user_results := entriesToUsers(&res.Entries)

	var user User
	for _, value := range user_results {
		s.db.Where("uid = ?", value.Uid).First(&user)

		if user.Uid == "" {
			s.db.Create(&value)
		} else {
			user.Refs = uint(float64(user.Refs) * 0.9)
			s.db.Save(&user)
		}
	}

	return user_results, nil
}

func entriesToUsers(entries *[]*ldap.Entry) Users {
	res := make(Users, len(*entries))

	for i, entry := range *entries {
		res[i] = User{
			Cn:      entry.GetAttributeValue("cn"),
			Uid:     entry.GetAttributeValue("uid"),
			UgKthid: entry.GetAttributeValue("ugKthid"),
		}
	}

	return res
}

func uniqueUsers(lists ...[]User) []User {
	m := make(map[string]User)

	for _, list := range lists {
		for _, user := range list {
			m[user.UgKthid] = user
		}

	}

	res := make(Users, 0, len(m))

	for _, user := range m {
		res = append(res, user)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Refs > res[j].Refs
	})

	return res
}
