package git

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

type ObjectType string

const (
	OBJECT_TYPE_UNKNOWN   ObjectType = "unknown"
	OBJECT_TYPE_TREE      ObjectType = "tree"
	OBJECT_TYPE_TAG       ObjectType = "tag"
	OBJECT_TYPE_COMMIT    ObjectType = "commit"
	OBJECT_TYPE_BLOB      ObjectType = "blob"
	OBJECT_TYPE_OFS_DELTA ObjectType = "offset_delta"
	OBJECT_TYPE_REF_DELTA ObjectType = "ref_delta"
)

type ObjectLocationType uint32

const (
	LOCATION_FILE  ObjectLocationType = 0
	LOCATION_PACK  ObjectLocationType = 1
	LOCATION_DELTA ObjectLocationType = 2
)

type Object struct {
	Hash            string
	LocationType    ObjectLocationType
	PackFile        string // only if LocationType == LOCATION_PACK
	Offset          int64  // only if LocationType == LOCATION_PACK
	Type            ObjectType
	Content         []byte
	ContentLen      int
	DeltaOffset     int64
	DeltaApplied    bool
	DeltaReference  string
	DeltaContent    []byte
	DeltaType       ObjectType
	DeltaContentLen int
}

func (o Object) String() string {
	return fmt.Sprintf("Object{Hash: %s, LocType: %s, Offset: %d, Type: %s, Len: %d}", o.Hash, o.GetLocationType(), o.Offset, o.GetType(), o.ContentLen)
}

func (o Object) GetLocationType() string {
	switch o.LocationType {
	case LOCATION_FILE:
		return "file"
	case LOCATION_PACK:
		return "pack"
	}

	return "unknown"
}

func (o Object) GetType() string {
	return string(o.Type)
}

// OpenObject returns a parsed object
// TODO: string? or another return type?
func (repo Repository) OpenObject(hash string) (Object, error) {
	var objectType ObjectType
	var objectLen int
	var deltaOffset int64
	var deltaRef string
	var objectBytes []byte
	var err error

	// log.Printf("OpenObject(%s)", hash)

	deltaOffset = 0

	object, ok := repo.Objects[hash]
	if !ok {
		return object, fmt.Errorf("could not find object with hash = %s", hash)
	}

	switch object.LocationType {
	case LOCATION_FILE:
		if objectType, objectLen, objectBytes, err = repo.OpenFileObject(object.Hash); err != nil {
			return Object{}, err
		}

	case LOCATION_PACK:
		if objectType, objectLen, deltaOffset, deltaRef, objectBytes, err = repo.OpenPackObject(object); err != nil {
			return Object{}, err
		}
	default:
		panic(fmt.Sprintf("unknown object location type: %d", object.LocationType))
	}

	// TODO: patch object instead of creating a new one
	return Object{
		Hash:           object.Hash,
		LocationType:   object.LocationType,
		PackFile:       object.PackFile,
		Offset:         object.Offset,
		Type:           objectType,
		Content:        objectBytes,
		ContentLen:     objectLen,
		DeltaOffset:    deltaOffset,
		DeltaReference: deltaRef,
		DeltaApplied:   false,
	}, nil
}

// OpenFileObject attemds to open an object file by its hash and returns its type, len, contents or an error
func (repo Repository) OpenFileObject(hash string) (ObjectType, int, []byte, error) {
	objectType := OBJECT_TYPE_UNKNOWN

	file, err := os.Open(path.Join(repo.GetObjectsDir(), hash[0:2], hash[2:]))
	if err != nil {
		return OBJECT_TYPE_UNKNOWN, 0, []byte{}, err
	}

	reader, err := zlib.NewReader(file)
	if err != nil {
		return OBJECT_TYPE_UNKNOWN, 0, []byte{}, err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return OBJECT_TYPE_UNKNOWN, 0, []byte{}, err
	}

	hashCheck := fmt.Sprintf("%x", sha1.Sum(data))
	if hashCheck != hash {
		return OBJECT_TYPE_UNKNOWN, 0, []byte{}, fmt.Errorf("invalid content hash")
	}

	idx := bytes.Index(data, []byte{0})
	header := data[:idx]

	parts := strings.Split(string(header), " ")
	contentSize, err := strconv.Atoi(parts[1])
	if err != nil {
		return OBJECT_TYPE_UNKNOWN, 0, []byte{}, err
	}

	if len(data[idx+1:]) != contentSize {
		return OBJECT_TYPE_UNKNOWN, 0, []byte{}, fmt.Errorf("invalid content size")
	}

	switch parts[0] {
	case "blob":
		objectType = OBJECT_TYPE_BLOB
	case "commit":
		objectType = OBJECT_TYPE_COMMIT
	case "tree":
		objectType = OBJECT_TYPE_TREE
	default:
		panic(fmt.Sprintf("unknown object type: %s", parts[0]))
	}

	return objectType, contentSize, data[idx+1:], nil
}
