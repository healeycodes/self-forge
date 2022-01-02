package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Lock a repository by name (also create mutex if required)
func lockRepo(repo string) {
	if m, found := repoMutexes[repo]; found {
		m.Lock()
	} else {
		m := &sync.Mutex{}
		m.Lock()
		repoMutexes[repo] = m
	}
}

// Unlock a repository by name
func unlockRepo(repo string) {
	if m, found := repoMutexes[repo]; found {
		m.Unlock()
	}
}

// Get all local repositories by name
func getAllLocalRepos() (map[string]bool, error) {
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
	target := path.Join(localGitPath, extra)

	// Files
	fileList := make([]gitFile, 0)
	files, err := ioutil.ReadDir(target)
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
	commitIter, err := r.Log(&git.LogOptions{Order: git.LogOrderCommitterTime})
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

// Get a commit for a given hash
func getCommit(repoName string, hash plumbing.Hash) (*object.Commit, error) {
	localGitPath := path.Join(repoPath, repoName)
	r, err := git.PlainOpen(localGitPath)
	if err != nil {
		return nil, err
	}

	commit, err := r.CommitObject(hash)
	if err != nil {
		return nil, err
	}

	return commit, nil
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
