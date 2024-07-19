package git

import (
	"fmt"
	"os"
	"path"
	"strings"
)

// List all known objects from "<repo>/.git/objects/??/*"
func (repo Repository) ListFileObjects() ([]Object, error) {
	knownObjects := make([]Object, 0)
	for n := range 256 {
		hashPart := fmt.Sprintf("%02x", n)

		dirPath := path.Join(repo.Path, ".git/objects", hashPart)
		stat, err := os.Stat(dirPath)
		if err != nil || !stat.IsDir() {
			continue
		}

		// List all objects
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return []Object{}, err
		}
		for _, entry := range entries {
			hash := hashPart + entry.Name()
			knownObjects = append(knownObjects, Object{
				Hash:         hash,
				LocationType: LOCATION_FILE,
			})
		}
	}

	return knownObjects, nil
}

// List all objects that can be extracted from "<repo>/.git/objects/pack/*"
func (repo Repository) ListPackedObjects() ([]Object, error) {
	knownObjects := make([]Object, 0)
	dirEntries, err := os.ReadDir(repo.GetPackDir())
	if err != nil {
		return []Object{}, fmt.Errorf("could not open pack directory: %s", repo.GetPackDir())
	}

	for _, dirEntry := range dirEntries {
		if !strings.HasSuffix(dirEntry.Name(), ".idx") {
			continue
		}

		idxDirEntry := dirEntry.Name()

		// check the .idx file has a .pack counterpart
		dirEntryBase := strings.TrimSuffix(idxDirEntry, ".idx")
		packDirEntry := dirEntryBase + ".pack"

		packPath := path.Join(repo.GetPackDir(), packDirEntry)
		_, err := os.Stat(packPath)
		if err != nil {
			return []Object{}, fmt.Errorf("could not stat pack: %s", packPath)
		}

		objects, err := repo.OpenPackIdx(idxDirEntry, packDirEntry)
		if err != nil {
			return []Object{}, err
		}

		for _, object := range objects {
			if object.Type == OBJECT_TYPE_UNKNOWN {
				continue
			}

			knownObjects = append(knownObjects, object)
		}
	}

	return knownObjects, nil
}

// ListObjects lists and returns both file & pack objects
func (repo Repository) ListObjects() (map[string]Object, error) {
	var fileObjects []Object
	var packedObjects []Object
	var err error

	objects := make(map[string]Object)

	if fileObjects, err = repo.ListFileObjects(); err != nil {
		return map[string]Object{}, err
	}

	for _, fileObject := range fileObjects {
		objects[fileObject.Hash] = fileObject
	}

	if packedObjects, err = repo.ListPackedObjects(); err != nil {
		return map[string]Object{}, err
	}

	for _, packObject := range packedObjects {
		objects[packObject.Hash] = packObject
	}

	repo.Objects = objects

	return objects, nil
}
