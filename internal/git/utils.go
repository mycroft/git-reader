package git

import (
	"bufio"
)

func ReadVariantInteger(reader *bufio.Reader, offset bool) int64 {
	var b byte
	var err error
	val := int64(0)

	for {
		if b, err = reader.ReadByte(); err != nil {
			panic(err)
		}

		val = (val << 7) | int64(b&0x7f)
		if b&0x80 == 0 {
			break
		}

		if offset {
			val += 1
		}
	}

	return val
}

func ReadVariantIntegerLE(reader *bufio.Reader) int64 {
	var b byte
	var err error
	val := int64(0)
	bshift := 0

	for {
		if b, err = reader.ReadByte(); err != nil {
			panic(err)
		}

		val |= int64(b&0x7f) << bshift
		if b&0x80 == 0 {
			break
		}

		bshift += 7
	}

	return val
}
