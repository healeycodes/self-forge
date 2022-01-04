package self_forge

import (
	"encoding/json"
	"fmt"
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

	_, found = os.LookupEnv("DEV")
	if found {
		return gitHubRepos{}, nil
	}

	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100", username)
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

func cloneRepo(repoName string, gitUrl string) (*git.Repository, error) {
	log.Printf("cloning: %s\n", gitUrl)
	r, err := git.PlainClone(repoPath+repoName, false, &git.CloneOptions{
		URL:      gitUrl,
		Progress: log.Writer(),
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

func updateRepos() error {
	log.Println("updating repositories")
	githubRepos, err := getGitHubInfo()
	if err != nil {
		return err
	}

	directories, err := GetAllLocalRepos()
	if err != nil {
		return err
	}

	for _, repo := range githubRepos {
		lockRepo(repo.Name)

		err = func() error {
			defer unlockRepo(repo.Name)

			var r *git.Repository
			var err error
			if _, ok := directories[repo.Name]; !ok {
				r, err = cloneRepo(repo.Name, repo.GitUrl)
				if err != nil {
					return err
				}
			} else {
				localGitPath := path.Join(repoPath, repo.Name)
				r, err = git.PlainOpen(localGitPath)
				if err != nil {
					return err
				}
			}

			err = r.Fetch(&git.FetchOptions{Force: true})
			if err != nil {
				if err != git.NoErrAlreadyUpToDate {
					return err
				}
			}

			w, err := r.Worktree()
			if err != nil {
				return err
			}

			log.Printf("pulling: %s\n", repo.Name)
			err = w.Pull(&git.PullOptions{Force: true})
			if err != nil {
				if err != git.NoErrAlreadyUpToDate {
					return err
				}
			}

			return nil
		}()

		if err != nil {
			return err
		}
	}
	return nil
}
