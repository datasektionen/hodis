package ldap

import (
	"github.com/datasektionen/hodis/server/models"

	"gopkg.in/ldap.v2"
)

func entriesToUsers(entries []*ldap.Entry) []models.User {
	res := make([]models.User, len(entries))
	for i, entry := range entries {
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
