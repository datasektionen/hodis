package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/jinzhu/gorm"
	"gopkg.in/ldap.v2"
)

type settings struct {
	queries *sync.Map

	ldap_host string
	ldap_port int

	base_dn    string
	attributes []string

	db *gorm.DB
}

var s settings

func LdapInit(ldap_host string, ldap_port int, base_dn string, db *gorm.DB) {
	s = settings{
		new(sync.Map),
		ldap_host,
		ldap_port,
		base_dn,
		[]string{"ugUsername", "ugKthid", "givenName", "displayName", "mail", "cn"},
		db,
	}
}



func Search(query string) (Users, error) {
	query = strings.ToLower(query)

	var db_results Users
	s.db.Where("uid = ? OR ug_kthid = ? OR LOWER(cn) LIKE ?", query, query, fmt.Sprintf("%%%s%%", query)).Find(&db_results)

	filter := fmt.Sprintf("(|(displayName=*%s*)(ugUsername=%s)(ugKthid=%s))", query, query, query)

	if _, ok := s.queries.Load(query); ok || len(db_results) >= 1000 {
		sort.Slice(db_results, func(i, j int) bool {
			return db_results[i].Refs > db_results[j].Refs
		})
		return db_results, nil
	}


	user_results, err := searchLDAP(filter)

	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %s", err.Error())
	}

	s.queries.Store(query, nil)

	return uniqueUsers(user_results, db_results), nil
}

func ExactUid(query string) (User, error) {
	return exactSearch(query, "ugUsername")
}

func ExactUgid(query string) (User, error) {
	return exactSearch(query, "ugKthid")
}

func exactSearch(query string, ldapField string) (User, error) {
	var user User
	userFound := false
	if ldapField == "ugUsername" {
		s.db.Where(User{Uid: query}).First(&user)
		userFound = user.Uid != ""
	} else if ldapField == "ugKthid" {
		s.db.Where(User{UgKthid: query}).First(&user)
		userFound = user.UgKthid != ""
	}

	if userFound {
		//Add an arbitrary ref to the user
		user.Refs++
		s.db.Save(&user)

		//Posibly fix missing fields
		//This should be removed in the future
		go searchLDAP(fmt.Sprintf("(%s=%s)", ldapField, query))
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

	//Update the db asynchronously
	go func() {
		for _, value := range user_results {
			user := User{Uid: ""}
			s.db.First(&user, "ug_kthid = ?", value.UgKthid)

			if user.Uid == "" {
				s.db.Create(&value)
			} else {
				//Add some arbitrary decay to the the users refs
				user.Refs = uint(float64(user.Refs) * 0.9)

				//Update names to match new base DN
				//These should be removed in the future
				user.Cn = value.Cn
				user.GivenName = value.GivenName
				user.DisplayName = value.DisplayName
				user.Mail = value.Mail

				s.db.Save(&user)
			}
		}
	}()

	return user_results, nil
}

func entriesToUsers(entries *[]*ldap.Entry) Users {
	res := make(Users, len(*entries))

	for i, entry := range *entries {
		res[i] = User{
			Cn: entry.GetAttributeValue("cn"),
			Uid: entry.GetAttributeValue("ugUsername"),
			UgKthid: entry.GetAttributeValue("ugKthid"),
			GivenName: entry.GetAttributeValue("givenName"),
			DisplayName: entry.GetAttributeValue("displayName"),
			Mail: entry.GetAttributeValue("mail"),
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
		if res[i].Year == res[j].Year {
			return res[i].Refs > res[j].Refs
		} else {
			return res[i].Year > res[j].Year
		}
	})

	return res
}
