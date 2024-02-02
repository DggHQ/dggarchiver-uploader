package md5

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

func strBin(s string) []uint32 {
	var i int32
	buf := make(map[int32]int32, 0)
	for i = 0; i < int32((len(s) << 3)); i += 8 {
		index1 := i >> 5
		index2 := i >> 3
		weow1 := 255 & int32(s[index2])
		weow2 := 31 & i
		weow := weow1 << weow2
		buf[index1] |= weow
	}
	res := make([]uint32, len(buf))
	for k, v := range buf {
		res[k] = uint32(v)
	}
	return res
}

func binHash(s []uint32) [4]int32 {
	var res [4]int32
	bytes := []byte{}

	for _, v := range s {
		bytesInner := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytesInner, v)
		bytes = append(bytes, bytesInner...)
	}

	hash := md5.Sum(bytes)
	for i := 0; i < 4; i++ {
		res[i] = int32(binary.LittleEndian.Uint32(hash[i*4 : (i*4)+4]))
	}

	return res
}

func binHex(hash [4]int32) string {
	var bytes []byte

	for _, v := range hash {
		bs := make([]byte, 4)
		binary.LittleEndian.PutUint32(bs, uint32(v))
		bytes = append(bytes, bs...)
	}

	return hex.EncodeToString(bytes)
}

func binHexBin(hash [4]int32) []uint32 {
	var res []uint32

	for _, v := range hash {
		bs := make([]byte, 4)
		binary.LittleEndian.PutUint32(bs, uint32(v))
		s := hex.EncodeToString(bs)
		res = append(res, strBin(s)...)
	}

	return res
}

func binHashStretch(s, salt string, length int) [4]int32 {
	e := fmt.Sprintf("%s%s", salt, s)
	// g := 32 + (int32(len(s)) << 3)
	o := strBin(s)
	a := len(o)
	resHashE := binHash(strBin(e))

	if length == 0 {
		length = 1024
	}

	for t := 0; t < length; t++ {
		e2 := binHexBin(resHashE)
		e2Map := make(map[int]uint32)
		for i, v := range e2 {
			e2Map[i] = v
		}
		for r := 0; r < a; r++ {
			e2Map[8+r] = o[r]
		}
		e2 = make([]uint32, len(e2Map))
		for i, v := range e2Map {
			e2[i] = v
		}
		resHashE = binHash(e2)
	}

	return resHashE
}

func HashStretch(s, salt string, length int) string {
	return binHex(binHashStretch(s, salt, length))
}

func Hash(s string) string {
	m := md5.Sum([]byte(s))
	return hex.EncodeToString(m[:])
}
