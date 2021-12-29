package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
)

const repoPath = "./repositories/"

func main() {
	if err := os.Mkdir(repoPath, 0755); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	repos, err := getRepos()
	if err != nil {
		log.Fatal(err)
	}

	err = cloneRepos(repos)
	if err != nil {
		log.Fatal(err)
	}
	serve()
}

func getRepos() (repos, error) {
	username, found := os.LookupEnv("GITHUB_USERNAME")
	if !found {
		log.Fatal("GITHUB_USERNAME not set")
	}

	perPage, found := os.LookupEnv("PER_PAGE")
	if !found {
		perPage = "5"
	}

	url := "https://api.github.com/users/" + username + "/repos?per_page=" + perPage
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var repos repos
	err = json.Unmarshal(b, &repos)
	if err != nil {
		return nil, err
	}

	return repos, nil
}

func cloneRepos(repos repos) error {
	start := time.Now()
	var wg sync.WaitGroup
	for _, repo := range repos {
		wg.Add(1)
		name := repo.Name
		gitUrl := repo.GitUrl
		go func() {
			defer wg.Done()
			cloneRepo(name, gitUrl)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("all clones done in %s", elapsed)

	return nil
}

func cloneRepo(name string, gitUrl string) {
	start := time.Now()
	log.Println("cloning: " + gitUrl)
	_, err := git.PlainClone(repoPath+name, false, &git.CloneOptions{
		URL:      gitUrl,
		Progress: log.Writer(),
	})
	if err != nil {
		log.Println(err)
	}
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func serve() {
	http.Handle("/", http.FileServer(http.Dir(repoPath)))
	port, found := os.LookupEnv("PORT")
	if !found {
		log.Fatal("PORT not set")
	}

	log.Printf("Serving on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

type repos []struct {
	GitUrl string `json:"git_url"`
	Name   string `json:"name"`
}
