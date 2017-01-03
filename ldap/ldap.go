package ldap

import (
	"fmt"
	"log"
	"strings"

	"gopkg.in/ldap.v2"
)

type User struct {
	Cn        string `json:"cn"`
	Uid       string `json:"uid"`
	UgKthid   string `json:"ugKthid"`
}

type Cache struct {
	Values  map[string]User
	Strings map[string]bool

	ldap_host string
	ldap_port int

	base_dn    string
	attributes []string
}

var cache Cache

func Init(ldap_host string, ldap_port int, base_dn string) {
	cache = Cache{
		make(map[string]User),
		make(map[string]bool),
		ldap_host,
		ldap_port,
		base_dn,
		[]string{"cn", "uid", "ugKthid"},
	}
}

func GetCache() []User {
	res := make([]User, len(cache.Values))
	i := 0
	for _, value := range cache.Values {
		res[i] = value
		i++
	}
	return res
}

func userToLower(user *User) {
	user.Cn = strings.ToLower(user.Cn)
	user.Uid = strings.ToLower(user.Uid)
	user.UgKthid = strings.ToLower(user.UgKthid)
}

func SearchWithCache(query User, exact bool) ([]User, error) {
	userToLower(&query)

	cache_results := make([]User, 0)
	for _, value := range cache.Values {
		if query.Cn != "" && strings.Contains(value.Cn, query.Cn) ||
			query.Uid != "" && strings.Contains(value.Uid, query.Uid) ||
			query.UgKthid != "" && strings.Contains(value.UgKthid, query.UgKthid) {

			cache_results = append(cache_results, value)

			if exact {
				return cache_results, nil
			}
		}
	}

	filter := makeFilter(query, exact)

	if len(cache_results) >= 1000 || cache.Strings[filter] {
		return cache_results, nil
	}

	user_results, err := searchLDAP(filter)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %s", err.Error())
	}

	for _, value := range user_results {
		cache.Values[value.Uid] = value
	}

	cache.Strings[filter] = true

	return user_results, nil
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

func makeFilter(query User, exact bool) string {
	filters := ""
	filters += addOption("cn", query.Cn, exact)
	filters += addOption("uid", query.Uid, exact)
	filters += addOption("ugKthid", query.UgKthid, exact)
	return fmt.Sprintf("(|%s)", filters)
}

func searchLDAP(filter string) ([]User, error) {
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", cache.ldap_host, cache.ldap_port))
	if err != nil {
		return nil, fmt.Errorf("Dial failed: %s", err.Error())
	}
	defer l.Close()

	res, err := l.SearchWithPaging(makeSearchRequest(filter), 10)
	if err != nil {
		if len(res.Entries) >= 0 {
			log.Printf("%s but %d results, continuing.", err.Error(), len(res.Entries))
		} else {
			return nil, fmt.Errorf("Paged search failed: %s", err.Error())
		}
	}

	return entryToUser(&res.Entries), nil
}

func makeSearchRequest(filter string) *ldap.SearchRequest {
	return ldap.NewSearchRequest(
		cache.base_dn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		cache.attributes,
		nil,
	)
}

func entryToUser(entries *[]*ldap.Entry) []User {
	res := make([]User, len(*entries))

	for i, entry := range *entries {
		res[i] = User{
			Cn:        entry.GetAttributeValue("cn"),
			Uid:       entry.GetAttributeValue("uid"),
			UgKthid:   entry.GetAttributeValue("ugKthid"),
		}
	}

	return res
}
