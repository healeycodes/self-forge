package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/healeycodes/self-forge/pkg/self_forge"
)

var (
	repositoryPath = "./repositories"
	testRepository = "a_repository"
	branchOne      = "one"
	branchTwo      = "two"
)

func TestMain(m *testing.M) {
	os.RemoveAll(repositoryPath)
	os.MkdirAll(repositoryPath, 0777)

	seedScript := `cd repositories
mkdir a_repository
cd a_repository
git init
git checkout -b one
echo "a" > a.txt 
git add .
git commit -m "Add a"
git checkout -b two
echo "b" > b.txt 
git add .
git commit -m "Add v"`

	parts := strings.Split(seedScript, "\n")
	cmd := exec.Command("sudo", "/bin/sh", "-c", strings.Join(parts, ";"))
	stdout, err := cmd.Output()
	fmt.Println(string(stdout))
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	code := m.Run()
	os.Exit(code)
}

func TestAPI(t *testing.T) {
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
