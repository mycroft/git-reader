package git

import (
	"fmt"
	"os"
	"path"
	"strings"
)

type Repository struct {
	Path    string
	Objects map[string]Object
}

// OpenRepository opens a repository
func OpenRepository(repopath string) (Repository, error) {
	// checks if the given path is a valid git repository
	stat, err := os.Stat(path.Join(repopath, ".git"))
	if err != nil {
		return Repository{}, err
	}

	if !stat.IsDir() {
		return Repository{}, fmt.Errorf("invalid repository path: %s", repopath)
	}

	repository := Repository{
		Path: repopath,
	}

	objects, err := repository.ListObjects()
	if err != nil {
		return Repository{}, err
	}

	repository.Objects = objects

	return repository, nil
}

func (repo Repository) GetObjectsDir() string {
	return path.Join(repo.Path, ".git/objects")
}

func (repo Repository) GetPackDir() string {
	return path.Join(repo.GetObjectsDir(), "pack")
}

func (repo Repository) GetCurrentRef() (string, error) {
	data, err := os.ReadFile(path.Join(repo.Path, ".git/HEAD"))
	if err != nil {
		return "", err
	}

	parts := strings.SplitN(strings.TrimSpace(string(data)), ": ", 2)

	data, err = os.ReadFile(path.Join(repo.Path, ".git", parts[1]))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}
