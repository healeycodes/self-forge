package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/gorilla/mux"
)

// - paginate GH API to get _all_ repos

const updateFrequency = 300 // seconds
const repoPath = "./repositories/"

var repoMutexes map[string]*sync.Mutex

func main() {
	repoMutexes = make(map[string]*sync.Mutex)
	if err := os.Mkdir(repoPath, 0755); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	// err := updateRepos()
	// if err != nil {
	// 	log.Printf("error: %s\n", err)
	// }

	// ticker := time.NewTicker(updateFrequency * time.Second)
	// go func() {
	// 	for {
	// 		<-ticker.C
	// 		err = updateRepos()
	// 		if err != nil {
	// 			log.Printf("error: %s\n", err)
	// 		}
	// 	}
	// }()

	serve()
}

func lockRepo(repo string) {
	if m, found := repoMutexes[repo]; found {
		m.Lock()
	} else {
		m := &sync.Mutex{}
		m.Lock()
		repoMutexes[repo] = m
	}
}

func unlockRepo(repo string) {
	if m, found := repoMutexes[repo]; found {
		m.Unlock()
	}
}

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

func allLocalRepos() (map[string]bool, error) {
	files, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}
	directories := make(map[string]bool)
	for _, file := range files {
		if file.IsDir() {
			directories[file.Name()] = true
		}
	}
	return directories, nil
}

// Get branches, files, and commits, for a repository and optional extra path
func getContext(repoName string, extra string) ([]string, []gitFile, []gitCommit, error) {
	localGitPath := path.Join(repoPath, repoName)
	if extra != "" {
		localGitPath = path.Join(localGitPath, extra)
	}

	// Files
	fileList := make([]gitFile, 0)
	files, err := ioutil.ReadDir(localGitPath)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, file := range files {
		if file.IsDir() && file.Name() == ".git" {
			continue
		}
		fileList = append(fileList, gitFile{
			Name:  file.Name(),
			IsDir: file.IsDir(),
		})
	}

	// Branches
	branchList := make([]string, 0)
	r, err := git.PlainOpen(localGitPath)
	if err != nil {
		return nil, nil, nil, err
	}

	refIter, _ := r.References()
	refIter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() {
			branchShort := strings.Replace(ref.Name().Short(), "origin/", "", 1)
			branchList = append(branchList, branchShort)
		}
		return nil
	})

	// Commits
	commitList := make([]gitCommit, 0)
	commitIter, err := r.Log(&git.LogOptions{})
	if err != nil {
		return nil, nil, nil, err
	}

	commitIter.ForEach(func(c *object.Commit) error {
		commitList = append(commitList, gitCommit{
			Hash:    c.Hash,
			Author:  c.Author.Name,
			Message: c.Message,
		})
		return nil
	})

	return branchList, fileList, commitList, nil
}

func renderContext(repository string, branch string, branchList []string, fileList []gitFile, commitList []gitCommit) string {
	ret := "<html>"

	// Branches
	ret += "<h3>branches</h3><ul>"
	for _, branchShort := range branchList {
		if branchShort == branch {
			ret += fmt.Sprintf("<li>%s</li>", branch)
		} else {
			ret += fmt.Sprintf("<li><a href=\"/%s/?branch=%s\">%s</a></li>", repository, branchShort, branchShort)
		}
	}
	ret += "</ul>"

	// Files
	ret += "<h3>files</h3><ul>"
	for _, gf := range fileList {
		var link string
		var text string
		if gf.IsDir {
			link = fmt.Sprintf("%s/%s/?branch=%s", repository, gf.Name, branch)
			text = fmt.Sprintf("<li><a href=\"/%s\">%s/</a></li>", link, gf.Name)
		} else {
			link = fmt.Sprintf("%s/%s?branch=%s", repository, gf.Name, branch)
			text = fmt.Sprintf("<li><a href=\"/%s\">%s</a></li>", link, gf.Name)
		}
		ret += text
	}
	ret += "</ul>"

	// Commits
	ret += "<h3>log</h3><ul>"
	for _, c := range commitList {
		ret += fmt.Sprintf("<li><a href=\"\">%s</a></li>", c.Message)
	}
	ret += "</ul>"

	return ret + "</html>"
}

// func renderCommit(repoName string, hash plumbing.Hash) (string, error) {
// 	localGitPath := path.Join(repoPath, repoName)
// 	r, err := git.PlainOpen(localGitPath)
// 	if err != nil {
// 		return "", err
// 	}

// 	c, err := r.CommitObject(hash)
// 	if err != nil {
// 		return "", err
// 	}

// 	println(c.String())
// 	return c.String(), nil
// }

func updateRepos() error {
	githubRepos, err := getGitHubInfo()
	if err != nil {
		return err
	}

	directories, err := allLocalRepos()
	if err != nil {
		return err
	}

	for _, repo := range githubRepos {
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
	}
	return nil
}

// Get branch from short name (or get default branch if branchShort is "")
func getBranchRefFromShort(repoName string, branchShort string) (plumbing.Hash, string, error) {
	localGitPath := path.Join(repoPath, repoName)
	r, err := git.PlainOpen(localGitPath)
	if err != nil {
		return plumbing.Hash{}, "", err
	}

	var refHash plumbing.Hash

	if branchShort == "" {
		branches, err := r.Branches()
		if err != nil {
			return plumbing.Hash{}, "", err
		}

		ref, err := branches.Next()
		if err != nil {
			return plumbing.Hash{}, "", err
		}

		head, err := r.Head()
		if err != nil {
			return plumbing.Hash{}, "", err
		}

		return head.Hash(), ref.Name().Short(), nil
	}

	refIter, _ := r.References()
	refIter.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsRemote() {
			if branchShort == strings.Replace(ref.Name().Short(), "origin/", "", 1) {
				refHash = ref.Hash()
			}
		}
		return nil
	})

	if refHash.IsZero() {
		return plumbing.Hash{}, "", fmt.Errorf("couldn't find branch: %s", branchShort)
	}

	return refHash, branchShort, nil
}

// Checkout a branch, return the branch short name
func checkoutBranch(repoName string, branchShort string) (string, error) {
	localGitPath := path.Join(repoPath, repoName)
	r, err := git.PlainOpen(localGitPath)
	if err != nil {
		return "", err
	}

	branchRef, branchShort, err := getBranchRefFromShort(repoName, branchShort)
	if err != nil {
		return "", err
	}

	w, err := r.Worktree()
	if err != nil {
		return "", err
	}

	err = w.Checkout(&git.CheckoutOptions{Hash: branchRef})
	if err != nil {
		return "", err
	}

	return branchShort, nil
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	directories, err := allLocalRepos()
	if err != nil {
		log.Printf("error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	list := make([]string, 0)
	for dir := range directories {
		list = append(list, dir)
	}
	json.NewEncoder(w).Encode(list)
}

func handleRepository(w http.ResponseWriter, r *http.Request) {
	repository := mux.Vars(r)["repository"]
	filePath := mux.Vars(r)["path"]
	branch := r.URL.Query().Get("branch")

	lockRepo(repository)
	defer unlockRepo(repository)

	currentBranchShort, err := checkoutBranch(repository, branch)
	if err != nil {
		log.Printf("error: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if strings.HasSuffix(filePath, "/") || filePath == "" {
		branchList, fileList, commitList, err := getContext(repository, filePath)
		if err != nil {
			log.Printf("error: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write([]byte(renderContext(path.Join(repository, filePath), currentBranchShort, branchList, fileList, commitList)))
	} else {
		http.ServeFile(w, r, path.Join(repoPath, repository, filePath))
	}
}

// func handleCommit(w http.ResponseWriter, r *http.Request) {
// 	repository := mux.Vars(r)["repository"]
// 	hash := mux.Vars(r)["hash"]

// 	lockRepo(repository)
// 	defer unlockRepo(repository)

// 	// TODO: checkout commit

// 	// TODO: render commit
// }

func serve() {
	port, found := os.LookupEnv("PORT")
	if !found {
		log.Fatal("PORT not set")
	}

	router := mux.NewRouter()
	router.HandleFunc("/", handleHome)
	router.HandleFunc("/{repository}/{path:.?|.+}", handleRepository)

	// TODO: rework links to use /tree/
	// router.HandleFunc("/{repository}/tree/{path:.?|.+}", handleRepoPath)

	// TODO: add commit handling
	// router.HandleFunc("/{repository}/commit/{path:.?|.+}", handleCommit)

	srv := &http.Server{
		Handler: router,
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
	}
	log.Fatal(srv.ListenAndServe())
}

type gitCommit struct {
	Hash    plumbing.Hash
	Author  string
	Message string
}

type gitFile struct {
	Name  string
	IsDir bool
}

type gitHubRepos []struct {
	GitUrl string `json:"git_url"`
	Name   string `json:"name"`
}
