package ldap

import (
	"fmt"
	"log"
	"strings"

	"../models"

	"gopkg.in/ldap.v2"
	"github.com/jinzhu/gorm"	
)

type Settings struct {
	queries map[string]bool

	ldap_host string
	ldap_port int

	base_dn    string
	attributes []string

	db         *gorm.DB
}

var s Settings

func Init(ldap_host string, ldap_port int, base_dn string, db *gorm.DB) {
	s = Settings{
		make(map[string]bool),
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

	if len(db_results) >= 1000 || s.queries[filter] {
		return db_results, nil
	}

	user_results, err := searchLDAP(filter)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %s", err.Error())
	}

	for _, value := range user_results {
		var users []models.User
		s.db.Where(models.User{Uid: value.Uid}).Find(&users)

		if len(users) == 0 {
			s.db.Create(&value)
		}
	}

	s.queries[filter] = true

	return uniqueUsers(user_results, db_results), nil
}

func addOption(field string, value string, exact bool) string {
	if !exact {
		value = fmt.Sprintf("*%s*", value)
	}
	if value != "" {
		return fmt.Sprintf("(%s=%s)", field, value)
	} else {
		return ""
	}
}

func makeFilter(query models.User, exact bool) string {
	filters := ""
	filters += addOption("cn", query.Cn, exact)
	filters += addOption("uid", query.Uid, false)
	filters += addOption("ugKthid", query.UgKthid, false)
	return fmt.Sprintf("(|%s)", filters)
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

func entriesToUsers(entries *[]*ldap.Entry) []models.User {
	res := make([]models.User, len(*entries))

	for i, entry := range *entries {
		res[i] = models.User{
			Cn:        entry.GetAttributeValue("cn"),
			Uid:       entry.GetAttributeValue("uid"),
			UgKthid:   entry.GetAttributeValue("ugKthid"),
		}
	}

	return res
}

func uniqueUsers(lists ...[]models.User) []models.User {
	m := make(map[string]models.User)

	for _, list := range lists {
		for _, value := range list {
			m[value.UgKthid] = value
		}
		
	}

	res := make([]models.User, 0, len(m))

	for  _, value := range m {
	   res = append(res, value)
	}

	return res
}
