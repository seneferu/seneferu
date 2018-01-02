# Repos

## `GET /repos`

Get all the repositories on the server.

* `Content-Type`: `"application/json"`
* `Accept`: `"application/json"`

===

### Example response

* `Status`: `200`
* `Content-Type`: `"application/json; charset=UTF-8"`

```
[{"org":"someorg","name":"TestRepo","url":"https://github.com/blabla/blabla"}]
```