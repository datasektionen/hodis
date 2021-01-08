package server

import (
	"sort"

	"github.com/datasektionen/hodis/server/models"
)

func uniqueUsers(lists ...[]models.User) []models.User {
	m := make(map[string]models.User)
	for _, list := range lists {
		for _, user := range list {
			m[user.UgKthid] = user
		}
	}

	res := make([]models.User, 0, len(m))
	for _, user := range m {
		res = append(res, user)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Year > res[j].Year
	})

	return res
}
