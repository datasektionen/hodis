package ldap

import (
	"fmt"
	"log"
	"strings"
	"sort"

	"../models"

	"gopkg.in/ldap.v2"
	"github.com/jinzhu/gorm"
)

type Settings struct {
	queries map[string]uint

	ldap_host string
	ldap_port int

	base_dn    string
	attributes []string

	db         *gorm.DB
}

var s Settings

func Init(ldap_host string, ldap_port int, base_dn string, db *gorm.DB) {
	s = Settings{
		make(map[string]uint),
		ldap_host,
		ldap_port,
		base_dn,
		[]string{"cn", "uid", "ugKthid"},
		db,
	}
}

func UserSearch(query string) (models.Users, error) {
	query = strings.ToLower(query)

	var db_results []models.User
    s.db.Where(models.User{Uid: query}).
            Or(models.User{UgKthid: query}).
            Or(models.User{Year: query}).
            Or("cn LIKE ?", fmt.Sprintf("%%%s%%", query)).
            Find(&db_results)

	filter := fmt.Sprintf("(|(cn =*%s*)(uid=*%s*)(ugKthid=*%s*)", query, query, query)

	return SearchWithDb(db_results, filter)
}

func ExactUid(query string) (models.User, error) {
	query = strings.ToLower(query)

	var db_results []models.User
	s.db.Where(models.User{Uid: query}).Find(&db_results)

	filter := fmt.Sprintf("(uid=%s)", query)

	users, err := SearchWithDb(db_results, filter)

	if err != nil {
		return models.User{}, err
	}

	if len(users) == 1 {
		return users[0], nil
	} else if len(users) > 1 {
		return models.User{}, fmt.Errorf("Exact search failed: Got multiple results.")
	} else {
		return models.User{}, fmt.Errorf("Exact search failed: No such user exists.")
	}
}

func ExactUgid(query string) (models.User, error) {
	query = strings.ToLower(query)

	var db_results models.Users
	s.db.Where(models.User{UgKthid: query}).Find(&db_results)

	filter := fmt.Sprintf("(ugKthid=%s)", query)

	users, err := SearchWithDb(db_results, filter)

	if err != nil {
		return models.User{}, err
	}

	if len(users) == 1 {
		return users[0], nil
	} else if len(users) > 1 {
		return models.User{}, fmt.Errorf("Exact search failed: Got multiple results.")
	} else {
		return models.User{}, fmt.Errorf("Exact search failed: No such user exists.")
	}
}

func SearchWithDb(db_results models.Users, filter string) (models.Users, error) {
	if len(db_results) < 5 {
		for _, user := range db_results {
			user.Refs++
			s.db.Save(&user)
		}
	}

	if len(db_results) >= 1000 || s.queries[filter] > 0 {
		return db_results, nil
	}

	user_results, err := searchLDAP(filter)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %s", err.Error())
	}

	for _, value := range user_results {
		var user models.User
		s.db.Where("uid = ?", value.Uid).First(&user)

		if users == nil {
			s.db.Create(&value)
		}
	}

	s.queries[filter]++
	return uniqueUsers(user_results, db_results), nil
}

func searchLDAP(filter string) ([]models.User, error) {
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", s.ldap_host, s.ldap_port))
	if err != nil {
		return nil, fmt.Errorf("Dial failed: %s", err.Error())
	}
	defer l.Close()

	res, err := l.SearchWithPaging(makeSearchRequest(filter), 10)
	if err != nil {
		if len(res.Entries) > 0 {
			log.Printf("%s but %d results, continuing.", err.Error(), len(res.Entries))
		} else {
			return nil, fmt.Errorf("Paged search failed: %s", err.Error())
		}
	}

	return entriesToUsers(&res.Entries), nil
}

func makeSearchRequest(filter string) *ldap.SearchRequest {
	return ldap.NewSearchRequest(
		s.base_dn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		s.attributes,
		nil,
	)
}

func entriesToUsers(entries *[]*ldap.Entry) models.Users {
	res := make(models.Users, len(*entries))

	for i, entry := range *entries {
		res[i] = models.User{
			Cn:      entry.GetAttributeValue("cn"),
			Uid:     entry.GetAttributeValue("uid"),
			UgKthid: entry.GetAttributeValue("ugKthid"),
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

	for  _, user := range m {
	   res = append(res, user)
	}

	sort.Sort(res)

	return res
}
