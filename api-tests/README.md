# api-tests

Black-box integration tests for the Avenue HTTP API. They exercise a real,
running server (and real database) over the network using `sdk.Client` —
the same typed client any other Go caller of the API would use — rather than
calling handler/db functions directly in-process.

## Running

Bring up a server to test against:

```sh
docker-compose up -d
```

Then run the tests:

```sh
go test ./api-tests/... -v
```

If no server is reachable at the configured base URL, every test skips
itself (rather than failing), so `go test ./...` stays safe to run in
environments without a server up.

## What's covered

- **auth_test.go** — login/logout, bad password rejection, the dashboard
  endpoint.
- **folder_test.go** — folder create/rename/trash/restore lifecycle, and
  that the listing endpoint sorts by name by default.
- **file_test.go** — file upload/rename/trash/restore lifecycle.
- **bulk_test.go** — bulk delete + bulk restore of a folder and a file
  together.

Each test creates its own randomly-named folders/files (`uniqueName(...)`)
and cleans up after itself (trash + permanently purge), so tests are safe to
run repeatedly against a shared account without accumulating junk or
colliding with each other.

## Env vars

| Variable | Default | Description |
| --- | --- | --- |
| `AVENUE_TEST_BASE_URL` | `http://localhost:8080` | Base URL of the server under test. |
| `AVENUE_TEST_EMAIL` / `AVENUE_TEST_PASSWORD` | *(none)* | Credentials to log in with. If unset and the server has `REGISTRATION_ENABLED=true` (docker-compose's dev config does), tests instead register a fresh throwaway account per run. If unset and registration is disabled, tests fall back to the seeded root user (`root@gmail.com` / `password`, or whatever `ROOT_USER_EMAIL`/`ROOT_USER_PASSWORD` were set to). |

## Adding tests

Use `authedClient(t)` to get a logged-in `*sdk.Client` and its auth header,
and `uniqueName(prefix)` to name anything you create so it can't collide with
other test runs or pre-existing data in the account. Clean up what you create
(trash + purge for folders/files) so repeated runs don't pile up state.
