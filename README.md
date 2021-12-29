# self-forge

One day, I'd like to write a lightweight clone of GitHub.

For now, here's < 100 lines of Go that host your source files.

- Clones all of a GitHub user's repositories
- Serves the default branch of each via `http.FileServer`

This is a good example use-case of `sync.WaitGroup` – all clones are run as concurrent goroutines.

```bash
PORT=":80" GITHUB_USERNAME="healeycodes" go run serve.go
# optionally use PER_PAGE to raise the number of repositories (up to 100)
# TODO: pagination for unlimited repositories
```
