package main

import (
	"errors"
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

	ldapHost string
	ldapPort int

	baseDn string

	db *gorm.DB
}

var s settings

func LdapInit(ldapHost string, ldapPort int, baseDn string, db *gorm.DB) {
	s = settings{
		queries:  new(sync.Map),
		ldapHost: ldapHost,
		ldapPort: ldapPort,
		baseDn:   baseDn,
		db:       db,
	}
}

var ldapAttributes = []string{"ugUsername", "ugKthid", "givenName", "displayName", "mail", "cn"}

func Search(query string) (Users, error) {
	query = strings.ToLower(query)

	var dbResults Users
	s.db.Where("uid = ? OR ug_kthid = ? OR LOWER(cn) LIKE ?", query, query, fmt.Sprintf("%%%s%%", query)).Find(&dbResults)

	filter := fmt.Sprintf("(|(displayName=*%s*)(ugUsername=%s)(ugKthid=%s))", query, query, query)

	if _, ok := s.queries.Load(query); ok || len(dbResults) >= 1000 {
		sort.Slice(dbResults, func(i, j int) bool {
			return dbResults[i].Refs > dbResults[j].Refs
		})
		return dbResults, nil
	}

	userResults, err := searchLDAP(filter)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %s", err.Error())
	}

	s.queries.Store(query, nil)

	return uniqueUsers(userResults, dbResults), nil
}

func ExactUid(query string) (User, error) {
	user, err := searchDB(User{Uid: query})
	if err != nil {
		return exactSearch(query, "ugUsername")
	}
	return user, nil
}

func ExactUgid(query string) (User, error) {
	user, err := searchDB(User{UgKthid: query})
	if err != nil {
		return exactSearch(query, "ugKthid")
	}
	return user, nil
}

func searchDB(u User) (User, error) {
	var user User
	s.db.Where(u).First(&user)
	if user.UgKthid == "" {
		return User{}, errors.New("no such user found")
	}
	user.Refs++
	s.db.Save(&user)
	return user, nil
}

func exactSearch(query string, ldapField string) (User, error) {
	users, err := searchLDAP(fmt.Sprintf("(%s=%s)", ldapField, query))
	if err != nil {
		return User{}, err
	}

	l := len(users)
	if l == 0 {
		return User{}, fmt.Errorf("exact search failed: No such user exists")
	} else if l > 1 {
		return User{}, fmt.Errorf("exact search failed: Got multiple results")
	}
	return users[0], nil
}

func searchLDAP(filter string) ([]User, error) {
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", s.ldapHost, s.ldapPort))
	if err != nil {
		return nil, fmt.Errorf("Dial failed: %s", err.Error())
	}
	defer l.Close()

	ldapSearchRequest := ldap.NewSearchRequest(
		s.baseDn,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		ldapAttributes,
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

	userResults := entriesToUsers(&res.Entries)

	// Update the db asynchronously
	go func() {
		for _, value := range userResults {
			var user User
			s.db.First(&user, "ug_kthid = ?", value.UgKthid)
			if user.Uid == "" {
				s.db.Create(&value)
			} else {
				// Add some arbitrary decay to the the users refs
				user.Refs = uint(float64(user.Refs) * 0.9)

				// Update names to match new base DN
				// These should be removed in the future
				user.Cn = value.Cn
				user.GivenName = value.GivenName
				user.DisplayName = value.DisplayName
				user.Mail = value.Mail

				s.db.Save(&user)
			}
		}
	}()

	return userResults, nil
}

func entriesToUsers(entries *[]*ldap.Entry) Users {
	res := make(Users, len(*entries))
	for i, entry := range *entries {
		res[i] = User{
			Cn:          entry.GetAttributeValue("cn"),
			Uid:         entry.GetAttributeValue("ugUsername"),
			UgKthid:     entry.GetAttributeValue("ugKthid"),
			GivenName:   entry.GetAttributeValue("givenName"),
			DisplayName: entry.GetAttributeValue("displayName"),
			Mail:        entry.GetAttributeValue("mail"),
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
		}
		return res[i].Year > res[j].Year
	})

	return res
}
