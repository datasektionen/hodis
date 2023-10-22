package ldap

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/datasektionen/hodis/models"

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

func Search(query string) (models.Users, error) {
	query = strings.ToLower(query)

	var dbResults models.Users
	s.db.Where("uid = ? OR ug_kthid = ? OR LOWER(cn) LIKE ?", query, query, fmt.Sprintf("%%%s%%", query)).Find(&dbResults)

	filter := fmt.Sprintf("(|(displayName=*%[1]s*)(ugUsername=%[1]s)(ugKthid=%[1]s))", ldap.EscapeFilter(query))

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

func ExactUid(query string) (models.User, error) {
	user, err := searchDB(models.User{Uid: query})
	if err != nil {
		return exactSearch(query, "ugUsername")
	}
	return user, nil
}

func ExactUgid(query string) (models.User, error) {
	user, err := searchDB(models.User{UgKthid: query})
	if err != nil {
		return exactSearch(query, "ugKthid")
	}
	return user, nil
}

func searchDB(u models.User) (models.User, error) {
	var user models.User
	s.db.Where(u).First(&user)
	if user.UgKthid == "" {
		return models.User{}, errors.New("no such user found")
	}
	user.Refs++
	s.db.Save(&user)
	return user, nil
}

func exactSearch(query string, ldapField string) (models.User, error) {
	users, err := searchLDAP(fmt.Sprintf("(%s=%s)", ldap.EscapeFilter(ldapField), ldap.EscapeFilter(query)))
	if err != nil {
		return models.User{}, err
	}

	l := len(users)
	if l == 0 {
		return models.User{}, fmt.Errorf("exact search failed: No such user exists")
	} else if l > 1 {
		return models.User{}, fmt.Errorf("exact search failed: Got multiple results")
	}
	return users[0], nil
}

func searchLDAP(filter string) ([]models.User, error) {
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
		for _, user := range userResults {
			if s.db.FirstOrCreate(&user, "ug_kthid = ?", user.UgKthid).RowsAffected == 0 {
				// Add some arbitrary decay to the the users refs
				user.Refs = uint(float64(user.Refs) * 0.9)

				s.db.Save(&user)
			}
		}
	}()

	return userResults, nil
}

func entriesToUsers(entries *[]*ldap.Entry) models.Users {
	res := make(models.Users, len(*entries))
	for i, entry := range *entries {
		res[i] = models.User{
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

func uniqueUsers(lists ...[]models.User) []models.User {
	m := make(map[string]models.User)
	for _, list := range lists {
		for _, user := range list {
			m[user.UgKthid] = user
		}
	}

	res := make(models.Users, 0, len(m))
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
