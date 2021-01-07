package pls

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func HasPermission(tokenType, tokenValue, permission string) bool {
	url := fmt.Sprintf("https://pls.datasektionen.se/api/%s/%s/hodis/%s", tokenType, tokenValue, permission)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if b, err := ioutil.ReadAll(resp.Body); err != nil || string(b) != "true" {
		return false
	}
	return true
}
