package git

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

const (
	OBJ_TYPE_FILE      = 100644
	OBJ_TYPE_EXEC      = 100755
	OBJ_TYPE_TREE      = 40000
	OBJ_TYPE_SYMLINK   = 120000
	OBJ_TYPE_SUBMODULE = 160000
)

type Blob struct {
	Hash  string
	Name  string
	Perms int
}

type Tree struct {
	Hash  string
	Name  string
	Trees map[string]Tree
	Blobs map[string]Blob
	Perms int
}

func (tree *Tree) String() string {
	out := ""

	for _, v := range tree.Trees {
		out += fmt.Sprintf("%06d tree %s   %s\n", v.Perms, v.Hash, v.Name)
	}

	for _, v := range tree.Blobs {
		out += fmt.Sprintf("%06d blob %s   %s\n", v.Perms, v.Hash, v.Name)
	}

	return out
}

func (repo Repository) ConvertTree(treeData []byte) (*Tree, error) {
	blobs := make(map[string]Blob)
	trees := make(map[string]Tree)

	reader := bufio.NewReader(bytes.NewReader(treeData))

	for {
		objectPermsAsStr, err := reader.ReadString(' ')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}

		objectPerms, err := strconv.Atoi(objectPermsAsStr[:len(objectPermsAsStr)-1])
		if err != nil {
			return nil, err
		}

		objectName, err := reader.ReadBytes(0)
		if err != nil {
			return nil, err
		}

		objectHash := make([]byte, HASH_SIZE)

		objectHashSize, err := reader.Read(objectHash)
		if err != nil {
			return nil, err
		}
		if objectHashSize != HASH_SIZE {
			return nil, fmt.Errorf("invalid hash size")
		}
		objectHashAsStr := fmt.Sprintf("%x", objectHash)

		if objectPerms == OBJ_TYPE_EXEC || objectPerms == OBJ_TYPE_FILE {
			blobs[objectHashAsStr] = Blob{
				Hash:  objectHashAsStr,
				Name:  string(objectName),
				Perms: objectPerms,
			}
		} else if objectPerms == OBJ_TYPE_TREE {
			trees[objectHashAsStr] = Tree{
				Hash:  objectHashAsStr,
				Name:  string(objectName),
				Perms: objectPerms,
			}
		}
	}

	return &Tree{
		Blobs: blobs,
		Trees: trees,
	}, nil
}
