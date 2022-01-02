package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/go-git/go-git/v5"
)

func getGitHubInfo() (gitHubRepos, error) {
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

	var repos gitHubRepos
	err = json.Unmarshal(b, &repos)
	if err != nil {
		log.Printf("%s\n", resp.Status)
		log.Printf("%s\n", url)
		log.Printf("%s\n", string(b))
		return nil, err
	}

	return repos, nil
}

func cloneRepo(repoName string, gitUrl string) error {
	_, err := git.PlainClone(repoPath+repoName, false, &git.CloneOptions{
		URL:      gitUrl,
		Progress: log.Writer(),
	})
	if err != nil {
		return err
	}

	return nil
}

func updateRepos() error {
	log.Println("updating repositories")
	githubRepos, err := getGitHubInfo()
	if err != nil {
		return err
	}

	directories, err := getAllLocalRepos()
	if err != nil {
		return err
	}

	for _, repo := range githubRepos {
		lockRepo(repo.Name)
		defer unlockRepo(repo.Name)

		if _, ok := directories[repo.Name]; !ok {
			log.Printf("cloning: %s\n", repo.GitUrl)
			err := cloneRepo(repo.Name, repo.GitUrl)
			if err != nil {
				return err
			}
		} else {
			localGitPath := path.Join(repoPath, repo.Name)
			log.Printf("opening: %s\n", localGitPath)
			r, err := git.PlainOpen(localGitPath)
			if err != nil {
				return err
			}

			log.Printf("fetching: %s\n", localGitPath)
			err = r.Fetch(&git.FetchOptions{Force: true})
			if err != nil {
				if err != git.NoErrAlreadyUpToDate {
					return err
				}
			}

			log.Printf("getting worktree: %s\n", localGitPath)
			w, err := r.Worktree()
			if err != nil {
				return err
			}

			log.Printf("pulling: %s\n", localGitPath)
			err = w.Pull(&git.PullOptions{Force: true})
			if err != nil {
				if err != git.NoErrAlreadyUpToDate {
					return err
				}
			}
		}
		unlockRepo(repo.Name)
	}
	return nil
}
