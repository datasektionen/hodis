package ldap

import (
	"fmt"
	"log"

	"github.com/datasektionen/hodis/server/models"
	"gopkg.in/ldap.v2"
)

type Ldap interface {
	Search(query string) ([]models.User, error)
	ExactUid(uid string) (models.User, error)
	ExactUgid(ugkthid string) (models.User, error)
}

type ldapImpl struct {
	LdapHost string
}

func New(ldapHost string) Ldap {
	return ldapImpl{LdapHost: ldapHost}
}

const baseDN = "ou=Addressbook,dc=kth,dc=se"

var ldapAttributes = []string{"ugUsername", "ugKthid", "givenName", "displayName", "mail", "cn"}

func (l ldapImpl) Search(query string) ([]models.User, error) {
	filter := fmt.Sprintf("(|(displayName=*%s*)(ugUsername=%s)(ugKthid=%s))", query, query, query)
	users, err := l.search(filter)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %v", err)
	}
	return users, nil
}

func (l ldapImpl) ExactUid(uid string) (models.User, error) {
	return l.exactSearch(uid, "ugUsername")
}

func (l ldapImpl) ExactUgid(ugkthid string) (models.User, error) {
	return l.exactSearch(ugkthid, "ugKthid")
}

func (l ldapImpl) exactSearch(query string, ldapField string) (models.User, error) {
	users, err := l.search(fmt.Sprintf("(%s=%s)", ldapField, query))
	if err != nil {
		return models.User{}, err
	}

	n := len(users)
	if n == 0 {
		return models.User{}, fmt.Errorf("exact search failed: No such user exists")
	} else if n > 1 {
		return models.User{}, fmt.Errorf("exact search failed: Got multiple results")
	}
	return users[0], nil
}

func (l ldapImpl) search(filter string) ([]models.User, error) {
	conn, err := ldap.Dial("tcp", l.LdapHost)
	if err != nil {
		return nil, fmt.Errorf("Dial failed: %v", err)
	}
	defer conn.Close()

	req := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		ldapAttributes,
		nil,
	)

	res, err := conn.SearchWithPaging(req, 10)
	if err != nil {
		if len(res.Entries) > 0 {
			log.Printf("%s but %d results, continuing.", err.Error(), len(res.Entries))
		} else {
			return nil, fmt.Errorf("Paged search failed: %s", err.Error())
		}
	}

	return entriesToUsers(res.Entries), nil
}
