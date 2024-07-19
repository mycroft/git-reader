package git

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
)

const (
	HASH_SIZE = 20
)

// OpenPackIdx opens an parse an packed .idx file
func (repo Repository) OpenPackIdx(idxDirEntry, packDirEntry string) ([]Object, error) {
	var fileFD *os.File
	var err error

	var entriesNum uint32
	offsets := make(map[string]int64)
	hashes := make([]string, 0)
	objects := make([]Object, 0)

	// ts_start := time.Now().UnixMilli()

	dirEntryPath := path.Join(repo.GetPackDir(), idxDirEntry)

	if fileFD, err = os.Open(dirEntryPath); err != nil {
		return objects, err
	}

	reader := bufio.NewReader(fileFD)

	header := make([]byte, 4)
	if _, err = reader.Read(header); err != nil {
		return objects, err
	}
	if !reflect.DeepEqual(header, []byte{255, 116, 79, 99}) {
		panic("Invalid header for a pack file")
	}

	version := make([]byte, 4)
	if _, err = reader.Read(version); err != nil {
		return objects, err
	}
	if !reflect.DeepEqual(version, []byte{0, 0, 0, 2}) {
		panic("Invalid version for a pack file")
	}

	for _ = range 256 {
		buf := make([]byte, 4)

		if _, err = reader.Read(buf); err != nil {
			return objects, err
		}

		newEntriesNum := binary.BigEndian.Uint32(buf)
		entriesNum = newEntriesNum
	}

	for _ = range entriesNum {
		buf := make([]byte, HASH_SIZE)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return objects, err
		}
		hashes = append(hashes, fmt.Sprintf("%x", buf))
	}

	// CRCs
	// for _ = range entriesNum {
	// 	crc := make([]byte, 4)
	// 	if _, err := io.ReadFull(reader, crc); err != nil {
	// 		return objects, err
	// 	}
	// }

	// Skipping reading all CRCs, as we're not using them
	reader.Discard(int(4 * entriesNum))

	// offsets
	for n := range entriesNum {
		offset := make([]byte, 4)
		if _, err := io.ReadFull(reader, offset); err != nil {
			return objects, err
		}
		offsets[hashes[n]] = int64(binary.BigEndian.Uint32(offset))

		if offsets[hashes[n]]&0x80000000 != 0 {
			panic("large offsets are not supported")
		}

		objects = append(objects, Object{
			Hash:         hashes[n],
			LocationType: LOCATION_PACK,
			PackFile:     packDirEntry,
			Offset:       offsets[hashes[n]],
		})
	}

	// log.Printf("parsed %s after %d ms\n", packDirEntry, time.Now().UnixMilli()-ts_start)

	return objects, nil
}

// ReadPackObject reads an object from given reader
func (repo Repository) ReadPackObject(fileFD *os.File, reader *bufio.Reader) (ObjectType, int, int64, string, []byte, error) {
	objectType := OBJECT_TYPE_UNKNOWN

	deltaOffset := int64(0)
	deltaReference := ""

	// extract type & size
	b, err := reader.ReadByte()
	if err != nil {
		return objectType, 0, 0, "", []byte{}, err
	}

	packedObjectType := int((b & 0x7f) >> 4)
	// 1: commit, 2: tree, 3: blob, 4: tag, 6: offset delta, 7: reference delta
	// OBJ_COMMIT, OBJ_TREE, OBJ_BLOB, OBJ_TAG, OBJ_OFS_DELTA, OBJ_REF_DELTA
	packedObjectSize := int(b & 0x0f)

	shift := 4
	for b&0x80 == 0x80 {
		b, err = reader.ReadByte()
		if err != nil {
			panic(err)
		}

		packedObjectSize |= int(b&0x7F) << shift
		shift += 7
	}

	switch packedObjectType {
	case 1:
		objectType = OBJECT_TYPE_COMMIT
	case 2:
		objectType = OBJECT_TYPE_TREE
	case 3:
		objectType = OBJECT_TYPE_BLOB
	case 4:
		objectType = OBJECT_TYPE_TAG
	case 6:
		objectType = OBJECT_TYPE_OFS_DELTA
	case 7:
		objectType = OBJECT_TYPE_REF_DELTA
	}

	// Note: In case of OBJECT_TYPE_OFS_DELTA, we need to store the offset,
	// before continuing reading stuff; for OBJECT_TYPE_REF_DELTA, it is
	// the reference of base object the delta will be applied on.
	if objectType == OBJECT_TYPE_OFS_DELTA {
		deltaOffset = ReadVariantInteger(reader, true)
	}

	if objectType == OBJECT_TYPE_REF_DELTA {
		data := make([]byte, 20)

		_, err := io.ReadFull(reader, data)
		if err != nil {
			return OBJECT_TYPE_UNKNOWN, 0, 0, "", []byte{}, err
		}

		deltaReference = fmt.Sprintf("%x", data)
	}

	zlibReader, err := zlib.NewReader(reader)
	if err != nil {
		return OBJECT_TYPE_UNKNOWN, 0, 0, "", []byte{}, err
	}

	data, err := io.ReadAll(zlibReader)
	if err != nil {
		return OBJECT_TYPE_UNKNOWN, 0, 0, "", []byte{}, err
	}

	if len(data) != int(packedObjectSize) {
		panic(fmt.Sprintf("error while parsing packed object: invalid size: %d != %d", len(data), packedObjectSize))
	}

	return objectType, len(data), deltaOffset, deltaReference, data, err
}

// OpenPackObject opens a pack file and attempts to retrieve a given object by its hash and offset
// It returns: Object's type (commit, blob...), object's size, object's contents or an error
func (repo Repository) OpenPackObject(object Object) (ObjectType, int, int64, string, []byte, error) {
	// To process an object embedded in a packfile, we must:
	// - move to correct offset
	// - extract object type
	// - extract compressed object size
	// - retrieve object's compressed data

	fileFD, err := os.Open(path.Join(repo.GetPackDir(), object.PackFile))
	if err != nil {
		panic(err)
	}

	fileFD.Seek(int64(object.Offset), 0)

	reader := bufio.NewReader(fileFD)

	return repo.ReadPackObject(fileFD, reader)
}

// ApplyDelta retrieves a offset_delta object, retrieves base object, patch base object content and returns the offset_delta patched
func (repo Repository) ApplyDelta(object Object) Object {
	var baseObjectHash string
	var baseObjectOffset int64

	if object.Type != OBJECT_TYPE_OFS_DELTA && object.Type != OBJECT_TYPE_REF_DELTA {
		return object
	}

	switch object.Type {
	case OBJECT_TYPE_OFS_DELTA:
		readOffset := object.DeltaOffset

		// We're in the same file. Offset returned is relative to current offset, and then we must return the object at
		// this area and so one
		baseObjectOffset = object.Offset - readOffset

		// Retrieve from the repository metadata teh correct commit
		// TODO: Optimize this
		for _, repoObject := range repo.Objects {
			if repoObject.PackFile != object.PackFile {
				continue
			}

			if repoObject.Offset == baseObjectOffset {
				baseObjectHash = repoObject.Hash
			}
		}

	case OBJECT_TYPE_REF_DELTA:
		baseObjectHash = object.DeltaReference
	}

	if baseObjectHash == "" {
		panic(fmt.Sprintf("could not find repoObject in %s at offset %d", object.PackFile, baseObjectOffset))
	}

	baseObject, err := repo.OpenObject(baseObjectHash)
	if err != nil {
		panic(err)
	}
	baseObject = repo.ApplyDelta(baseObject)

	// Transform the data
	transformReader := bufio.NewReader(bytes.NewReader(object.Content))
	_ = ReadVariantIntegerLE(transformReader) // baseObjSize
	_ = ReadVariantIntegerLE(transformReader) // ObjSizeDest

	destObject := make([]byte, 0)

	for {
		ch, err := transformReader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic(err)
			}
		}

		if ch == 0x80 {
			continue
		}

		if (ch & 0x80) != 0 {
			// copy data from base object
			vals := make([]byte, 0)

			for i := range 7 {
				bmask := byte(1 << i)
				if ch&bmask != 0 {
					nc, err := transformReader.ReadByte()
					if err != nil {
						panic(err)
					}
					vals = append(vals, nc)
				} else {
					vals = append(vals, 0)
				}
			}

			start := binary.LittleEndian.Uint32(vals[0:4])
			nbytes := uint32(binary.LittleEndian.Uint16(vals[4:6]))
			if nbytes == 0 {
				nbytes = 0x10000
			}

			// log.Printf("COPY FROM BASE OBJECT: start=0x%x, #bytes=%d", start, nbytes)
			destObject = append(destObject, baseObject.Content[start:start+nbytes]...)

		} else {
			// add new data
			nbytes := ch & 0x7f

			// log.Printf("APPEND NEW BYTES: #bytes=%d", nbytes)

			nBytesData := make([]byte, nbytes)
			_, err := transformReader.Read(nBytesData)
			if err != nil {
				panic(err)
			}

			destObject = append(destObject, nBytesData...)
		}
	}

	// fmt.Println(len(destObject), ObjSizeDest, baseObjSize)

	return Object{
		Hash:            object.Hash,
		LocationType:    object.LocationType,
		Offset:          object.Offset,
		Type:            baseObject.Type,
		Content:         destObject,
		ContentLen:      baseObject.ContentLen,
		DeltaApplied:    true,
		DeltaType:       object.Type,
		DeltaContent:    object.Content,
		DeltaContentLen: object.ContentLen,
	}
}

func (repo Repository) ReadObjRefDeltaObject() {

}
