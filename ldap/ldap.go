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

func userToLower(user *models.User) {
	user.Cn = strings.ToLower(user.Cn)
	user.Uid = strings.ToLower(user.Uid)
	user.UgKthid = strings.ToLower(user.UgKthid)
}

func SearchWithDb(query models.User, exact bool) ([]models.User, error) {
	userToLower(&query)

	var db_results []models.User
	s.db.Where(models.User{Uid: query.Uid}).
		Or(models.User{UgKthid: query.UgKthid}).
		Or(models.User{Year: query.Year}).
		Or("cn LIKE ?", fmt.Sprintf("%%%s%%", query.Cn)).
		Find(&db_results)

	filter := makeFilter(query, exact)

	s.queries[filter]++
	if s.queries[filter] % 5 == 0 {
		var user models.User
		s.db.Where(models.User{Uid: query.Uid}).First(&user)
		s.db.Model(&user).Update("refs", user.Refs + 5)
	}

	if len(db_results) >= 1000 || s.queries[filter] > 0 {
		return db_results, nil
	}

	user_results, err := searchLDAP(filter)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %s", err.Error())
	}

	for _, value := range user_results {
		var users models.Users
		s.db.Where(models.User{Uid: value.Uid}).Find(&users)

		if len(users) == 0 {
			s.db.Create(&value)
		}
	}


	return uniqueUsers(user_results, db_results), nil
}

func addOption(field string, value string, exact bool) string {
	if value != "" {
		if !exact {
			value = fmt.Sprintf("*%s*", value)
		}
		return fmt.Sprintf("(%s=%s)", field, value)
	} else {
		return ""
	}
}

func makeFilter(query models.User, exact bool) string {
	filters := "(|"
	filters += addOption("cn", query.Cn, exact)
	filters += addOption("uid", query.Uid, exact)
	filters += addOption("ugKthid", query.UgKthid, exact)
	filters += ")"
	return filters
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
