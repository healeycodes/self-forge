package self_forge

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/gorilla/mux"
)

var (
	updateFrequency time.Duration = 3600
	repoPath        string        = "./repositories/"
	repoMutexes     map[string]*sync.Mutex
)

func Start() {
	repoMutexes = make(map[string]*sync.Mutex)

	if err := os.Mkdir(repoPath, 0755); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}

	err := updateRepos()
	if err != nil {
		log.Printf("error: %s\n", err)
	}

	ticker := time.NewTicker(updateFrequency * time.Second)
	go func() {
		for {
			<-ticker.C
			err = updateRepos()
			if err != nil {
				log.Printf("error: %s\n", err)
			}
		}
	}()

	serve()
}

func serve() {
	port, found := os.LookupEnv("PORT")
	if !found {
		port = "80"
	}

	router := mux.NewRouter()
	router.HandleFunc("/", handleHome)
	router.HandleFunc("/{repository}/tree/{path:.?|.+}", handleRepository)
	router.HandleFunc("/{repository}/commit/{hash}", handleCommit)

	srv := &http.Server{
		Handler: router,
		Addr:    fmt.Sprintf(":%s", port),
	}
	log.Fatal(srv.ListenAndServe())
}

func handleError(w http.ResponseWriter, err error) {
	log.Printf("error: %s", err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	directories, err := GetAllLocalRepos()
	if err != nil {
		handleError(w, err)
		return
	}

	list := make([]string, 0)
	for dir := range directories {
		list = append(list, dir)
	}

	username, _ := os.LookupEnv("GITHUB_USERNAME")
	w.Write([]byte(renderHome(username, list)))
}

func handleRepository(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	repository := mux.Vars(r)["repository"]
	filePath := mux.Vars(r)["path"]
	branch := r.URL.Query().Get("branch")

	lockRepo(repository)
	defer unlockRepo(repository)

	branchShort, err := CheckoutBranch(repository, branch)
	if err != nil {
		handleError(w, err)
		return
	}

	if strings.HasSuffix(filePath, "/") || filePath == "" {
		branchList, fileList, commitList, err := GetContext(repository, filePath)
		if err != nil {
			handleError(w, err)
			return
		}

		w.Write([]byte(renderContext(repository, filePath, branchShort, branchList, fileList, commitList)))
	} else {
		http.ServeFile(w, r, path.Join(repoPath, repository, filePath))
	}
}

func handleCommit(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	repository := mux.Vars(r)["repository"]
	hash := mux.Vars(r)["hash"]

	lockRepo(repository)
	defer unlockRepo(repository)

	if len(hash) != 40 {
		err := fmt.Errorf("hash incorrect length %s", hash)
		handleError(w, err)
		return
	}

	commit, err := getCommit(repository, plumbing.NewHash(hash))
	if err != nil {
		handleError(w, err)
		return
	}

	renderedCommit, err := renderCommit(commit)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Write([]byte(renderedCommit))
}
