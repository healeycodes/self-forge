# self-forge

_(Under construction)_

Self-host your GitHub repositories and serve them with a lightweight user interface (no JS or CSS).

## TODO

- Add server-rendered method to highlight lines of code.
- Add pagination to the GitHub API call to get all public repositories (either via library or manually). This reduces the limit from 100 to all.
- Add e2e tests (currently there's some integration tests).

## Run

- `PORT` serve from 0.0.0.0:PORT
- `LIMIT_REPOS` pull a limited amount of repositories
- `GITHUB_USERNAME` use this account's public repositories

```bash
PORT="80" LIMIT_REPOS=5 GITHUB_USERNAME="healeycodes" go run ./cmd
```

## Test

```bash
go test -v ./test
```