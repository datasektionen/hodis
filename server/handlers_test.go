package server_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/datasektionen/hodis/server"
	"github.com/datasektionen/hodis/server/database"
	"github.com/datasektionen/hodis/server/ldap"
	"github.com/datasektionen/hodis/server/models"

	"github.com/google/go-cmp/cmp"
)

func TestPing(t *testing.T) {
	r := (&server.Server{}).Router()
	req := httptest.NewRequest("GET", "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("resp.StatusCode got: %v want: 200", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error while reading response body: %v", err)
	}
	if got, want := string(b), `"Pong"`; got != want {
		t.Errorf("response body got: %q want: %q", got, want)
	}
}

var (
	tomten = models.User{
		UgKthid:     "u1xxxxx",
		Uid:         "tomten",
		Cn:          "Tomten Andersson (tomten)",
		Mail:        "tomten@kth.se",
		GivenName:   "Tomten",
		DisplayName: "Tomten Andersson",
		Year:        2020,
		Tag:         "foo",
	}
	tandfen = models.User{
		UgKthid:     "u1yyyyy",
		Uid:         "tandfen",
		Cn:          "Tandfen Larsson (tandfen)",
		Mail:        "tandfen@kth.se",
		GivenName:   "Tandfen",
		DisplayName: "Tandfen Larsson",
		Year:        2019,
		Tag:         "bar",
	}
)

type fantasyDatabase struct{}

func (fantasyDatabase) AllUsers() []models.User {
	return []models.User{tomten, tandfen}
}

func (fantasyDatabase) UserFromTag(tag string) models.User {
	return tomten
}

func (fantasyDatabase) Search(term string) []models.User {
	return []models.User{tomten, tandfen}
}

func (fantasyDatabase) ExactUID(uid string) models.User {
	return tomten
}

func (fantasyDatabase) ExactUgKthID(ugkthid string) models.User {
	return tomten
}

func (fantasyDatabase) AddUsers(users ...models.User) {}

func (fantasyDatabase) CreateOrUpdate(uid string, newUser models.User) models.User {
	return tomten
}

type emptyDatabase struct{}

func (emptyDatabase) AllUsers() []models.User {
	return []models.User{}
}

func (emptyDatabase) UserFromTag(tag string) models.User {
	return models.User{}
}

func (emptyDatabase) Search(term string) []models.User {
	return []models.User{}
}

func (emptyDatabase) ExactUID(uid string) models.User {
	return models.User{}
}

func (emptyDatabase) ExactUgKthID(ugkthid string) models.User {
	return models.User{}
}

func (emptyDatabase) AddUsers(users ...models.User) {}

func (emptyDatabase) CreateOrUpdate(uid string, newUser models.User) models.User {
	return models.User{}
}

func TestTag(t *testing.T) {
	r := (&server.Server{DB: fantasyDatabase{}}).Router()
	req := httptest.NewRequest("GET", "/tag/foo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("resp.StatusCode got: %v want: 200", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error while reading response body: %v", err)
	}
	var user models.User
	if err := json.Unmarshal(b, &user); err != nil {
		t.Fatalf("error while decoding response body: %v", err)
	}
	if diff := cmp.Diff(user, tomten); diff != "" {
		t.Errorf("invalid user returned diff (-got, +want):\n%v", diff)
	}
}

func TestSearchError(t *testing.T) {
	tests := []struct {
		name string
		path string
		db   database.Database
		ldap ldap.Ldap
		want int
	}{
		{
			name: "UserSearch not found",
			path: "/users/foo",
			db:   emptyDatabase{},
			ldap: errorLdap{},
			want: 501,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := (&server.Server{DB: tt.db, Ldap: tt.ldap}).Router()
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			resp := rec.Result()
			if resp.StatusCode != tt.want {
				t.Errorf("resp.StatusCode got: %v want: %v", resp.StatusCode, tt.want)
			}
		})
	}
}

func TestTagNotFound(t *testing.T) {
	r := (&server.Server{DB: emptyDatabase{}}).Router()
	req := httptest.NewRequest("GET", "/tag/foo", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if resp := rec.Result(); resp.StatusCode != 404 {
		t.Errorf("resp.StatusCode got: %v want: 404", resp.StatusCode)
	}
}

func TestCache(t *testing.T) {
	r := (&server.Server{DB: fantasyDatabase{}}).Router()
	req := httptest.NewRequest("GET", "/cache", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("resp.StatusCode got: %v want: 200", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error while reading response body: %v", err)
	}

	type response struct {
		Cache  []models.User `json:"cache"`
		Length int           `json:"length"`
	}
	got := response{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("error while decoding response body: %v", err)
	}
	want := response{
		Cache:  []models.User{tomten, tandfen},
		Length: 2,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("invalid user returned diff (-got, +want):\n%v", diff)
	}
}

var (
	paskharen = models.User{
		UgKthid:     "u1zzzzz",
		Uid:         "paskharen",
		Cn:          "Paskharen Olofsson (paskharen)",
		Mail:        "paskharen@kth.se",
		GivenName:   "Paskharen",
		DisplayName: "Paskharen Olofsson",
		Year:        2018,
		Tag:         "abc",
	}
)

type fantasyLdap struct{}

func (fantasyLdap) Search(query string) ([]models.User, error) {
	return []models.User{paskharen, tandfen}, nil
}

func (fantasyLdap) ExactUid(uid string) (models.User, error) {
	return paskharen, nil
}

func (fantasyLdap) ExactUgid(ugkthid string) (models.User, error) {
	return paskharen, nil
}

type errorLdap struct{}

func (errorLdap) Search(query string) ([]models.User, error) {
	return nil, errors.New("error")
}

func (errorLdap) ExactUid(uid string) (models.User, error) {
	return models.User{}, errors.New("error")
}

func (errorLdap) ExactUgid(ugkthid string) (models.User, error) {
	return models.User{}, errors.New("error")
}

type largeDatabase struct{}

func (largeDatabase) AllUsers() []models.User {
	return []models.User{}
}

func (largeDatabase) UserFromTag(tag string) models.User {
	return models.User{}
}

func (largeDatabase) Search(term string) []models.User {
	var users []models.User
	for i := 0; i < 1010; i++ {
		users = append(users, tomten)
	}
	return users
}

func (largeDatabase) ExactUID(uid string) models.User {
	return models.User{}
}

func (largeDatabase) ExactUgKthID(ugkthid string) models.User {
	return models.User{}
}

func (largeDatabase) AddUsers(users ...models.User) {}

func (largeDatabase) CreateOrUpdate(uid string, newUser models.User) models.User {
	return models.User{}
}

func TestUserSearch(t *testing.T) {
	tests := []struct {
		name string
		db   database.Database
		ldap ldap.Ldap
		want []models.User
	}{
		{
			name: "Both database and ldap results",
			db:   fantasyDatabase{},
			ldap: fantasyLdap{},
			want: []models.User{tomten, tandfen, paskharen},
		},
		{
			name: "Only ldap results",
			db:   emptyDatabase{},
			ldap: fantasyLdap{},
			want: []models.User{tandfen, paskharen},
		},
		{
			name: "Large database",
			db:   largeDatabase{},
			ldap: errorLdap{},
			want: func() []models.User {
				var res []models.User
				for i := 0; i < 1010; i++ {
					res = append(res, tomten)
				}
				return res
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := (&server.Server{DB: tt.db, Ldap: tt.ldap}).Router()
			req := httptest.NewRequest("GET", "/users/fooquery", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			resp := rec.Result()
			if resp.StatusCode != 200 {
				t.Errorf("resp.StatusCode got: %v want: 200", resp.StatusCode)
			}

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("error while reading response body: %v", err)
			}
			var users []models.User
			if err := json.Unmarshal(b, &users); err != nil {
				t.Fatalf("error while decoding response body: %v", err)
			}
			if diff := cmp.Diff(users, tt.want); diff != "" {
				t.Errorf("invalid user returned diff (-got, +want):\n%v", diff)
			}
		})
	}
}

func TestUserSearchCachedQuery(t *testing.T) {
	r := (&server.Server{DB: fantasyDatabase{}, Ldap: fantasyLdap{}}).Router()
	// Send first request
	req := httptest.NewRequest("GET", "/users/fooquery", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("resp.StatusCode got: %v want: 200", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error while reading response body: %v", err)
	}
	var users []models.User
	if err := json.Unmarshal(b, &users); err != nil {
		t.Fatalf("error while decoding response body: %v", err)
	}
	want := []models.User{tomten, tandfen, paskharen}
	if diff := cmp.Diff(users, want); diff != "" {
		t.Errorf("invalid user returned diff (-got, +want):\n%v", diff)
	}

	// Second request
	req = httptest.NewRequest("GET", "/users/fooquery", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp = rec.Result()
	if resp.StatusCode != 200 {
		t.Errorf("resp.StatusCode got: %v want: 200", resp.StatusCode)
	}

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error while reading response body: %v", err)
	}
	if err := json.Unmarshal(b, &users); err != nil {
		t.Fatalf("error while decoding response body: %v", err)
	}
	want = []models.User{tomten, tandfen}
	if diff := cmp.Diff(users, want); diff != "" {
		t.Errorf("invalid user returned diff (-got, +want):\n%v", diff)
	}
}
