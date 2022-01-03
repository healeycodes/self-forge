package self_forge

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

var (
	repoPath = "repositories"
)

func TestMain(m *testing.M) {
	os.RemoveAll(repoPath)
	os.MkdirAll(repoPath, 0755)
	code := m.Run()
	os.Exit(code)
}

func TestAPI(t *testing.T) {

	cmd := exec.Command("/bin/sh", "./seed_test_repository.sh")
	stdout, err := cmd.Output()
	fmt.Println(stdout)

	if err != nil {
		t.Error(err)
	}

	repositories, err := getAllLocalRepos()
	if err != nil {
		t.Error(err)
	}
	assertEqual(repositories[0], "a_repository")
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
