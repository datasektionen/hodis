# Hodis - User lookup system

## API

### GET /ping

Returns "Pong". Can be used for health-checking of the service.

### GET /users/:query

Searches kthid, ugkthid and name for `:query`. Responds with a list of users. Example:

```json
[
    {
        "ugKthid": "u1xxxxxx",
        "uid": "tomten",
        "cn": "Tomten Andersson (tomten)",
        "mail": "tomten@kth.se",
        "givenName": "Tomten",
        "displayName": "Tomten Andersson",
        "year": 2018,
        "tag":""
    },
    {
        "ugKthid": "u1xxxxxx",
        "uid": "tandfen",
        "cn": "Tandfen Persson (tandfen)",
        "mail": "tandfen@kth.se",
        "givenName": "Tandfen",
        "displayName": "Tandfen Persson",
        "year": 2017,
        "tag":""
    }
]
```

### GET /uid/:uid

Searches kthid for `:uid`. Responds with exactly one user. Same format as above.

### GET /ugkthid/:ugid

Searches ugkthid for `:ugid`. Responds with exactly one user. Same format as above.

### GET /tag/:tag

Searches for the first user with the given `:tag`. Tags seem to have been an experimental feature that isn't used. Responds with exactly one user. Same format as above.

### POST /uid/:uid

Updates a user. The user details can be specified either as a json body or as form fields. The format is the same as above.

Authentication is done by specifying an `api_key` or a `token`. The API key is a PLS API key and the token is a Login token. Users can modify their own profiles. If a user has the hodis.admin permission in PLS then they can modify other users profiles.

## Environment variables

| Variable      | Description                               | Example                               |
|---------------|-------------------------------------------|---------------------------------------|
| LOGIN_API_KEY | API key for the Login system.             | --                                    |
| LOGIN_URL     | URL to the Login system.                  | https://login.datasektionen.se        |
| LDAP_HOST     | hostname to kth:s ldap server.            | ldap.kth.se                           |
| LDAP_PORT     | tcp to kth:s ldap server.                 | 389                                   |
| DATABASE_URL  | A postgresql database url.                | postgres://postgres:password@db:5432/ |
| GIN_MODE      | Should be set to "release" in production. | |

## Dependency on other systems at Datasektionen

- **Login**: Used to authenticate users before changing their profiles.
- **PLS**: Used to check permissions before changing users profiles.

## PLS permissions

- hodis.admin: Allow API Keys or users to change other users profiles.

## Production setup

Set environment variables according to "Environment variables", install go, run `go build hodis.go` and start the server with `./hodis`.

## Development setup

Make `DATABASE_URL` is corretly pointing to a postgres database. You can start one with:
```sh
docker run \
    --name hodis-db -p 5432:5432 -d \
    -e POSTGRES_PASSWORD=hodis \
    -e POSTGRES_DB=hodis \
    -e POSTGRES_USER=hodis \
    postgres
```
and `DATABASE_URL=postgres://hodis:hodis@localhost:5432/hodis?sslmode=disable`

If you're at KTH, ldap should just work. Otherwise, create a tcp tunnel through
mjukglass using something like: `ssh mjukglass -L 33389:ldap.kth.se:389` and set
`LDAP_HOST` to localhost and `LDAP_PORT` to 389.
