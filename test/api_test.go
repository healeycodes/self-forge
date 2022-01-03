package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"testing"

	"github.com/healeycodes/self-forge/pkg/self_forge"
)

var (
	seedScript     = "./seed_test_repository.sh"
	repositoryPath = "./repositories"
	testRepository = "a_repository"
	branchOne      = "one"
	branchTwo      = "two"
)

func TestMain(m *testing.M) {
	os.RemoveAll(repositoryPath)
	os.MkdirAll(repositoryPath, 0755)
	code := m.Run()
	os.Exit(code)
}

func TestAPI(t *testing.T) {

	cmd := exec.Command("/bin/sh", seedScript)
	stdout, err := cmd.Output()
	fmt.Println(string(stdout))

	if err != nil {
		t.Error(err)
	}

	repositories, err := self_forge.GetAllLocalRepos()
	if err != nil {
		t.Error(err)
	}
	for name := range repositories {
		assertEqual(t, name, testRepository)
	}

	branchShort, err := self_forge.CheckoutBranch(testRepository, branchOne)
	if err != nil {
		t.Error(err)
	}
	assertEqual(t, branchShort, branchOne)

	branchList, fileList, commitList, err := self_forge.GetContext(testRepository, "")
	if err != nil {
		t.Error(err)
	}

	assertEqual(t, len(branchList), 2)
	sort.Strings(branchList)
	assertEqual(t, branchList[0], branchOne)
	assertEqual(t, branchList[1], branchTwo)

	for _, gf := range fileList {
		assertEqual(t, gf.Name, "a.txt")
		assertEqual(t, gf.IsDir, false)
	}

	for _, c := range commitList {
		assertEqual(t, c.Message, "Add a\n")
	}
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
