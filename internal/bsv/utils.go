package bsv

import (
	"bytes"
	"hash"
	"io"

	"github.com/coldstar-507/utils/utils"
	// "golang.org/x/crypto/ripemd160"
)

func StrippedAdress(checkAdress []byte) []byte {
	return checkAdress[1 : len(checkAdress)-4]
}

func StrippedCheck(checkAdress []byte, sh256 hash.Hash, sum []byte) ([]byte, bool) {
	ext := checkAdress[:len(checkAdress)-4]
	checksum := checkAdress[len(checkAdress)-4:]
	check := doubleHash(ext, sh256, sum)
	valid := bytes.Equal(check[:4], checksum)
	return ext[1:], valid
}

func RawAdressw(w io.Writer, pubKey []byte, sh256, rm hash.Hash, sum []byte) {
	sh256.Reset()
	rm.Reset()
	sh256.Write(pubKey)
	sum = sh256.Sum(sum)
	rm.Write(sum)
	utils.WriteBin(w, rm.Sum(sum[:0]))
}

func MakeRawAdress(pubKey []byte, sh256, rm hash.Hash, sum []byte) [20]byte {
	array := [20]byte{}
	buf := bytes.NewBuffer(array[:0])
	RawAdressw(buf, pubKey, sh256, rm, sum)
	return array
}

func MakeSecret(s1 []byte, ix uint32, h hash.Hash) []byte {
	h.Reset()
	utils.WriteBin(h, ix, s1)
	return h.Sum(nil)
}

func doubleHash(data []byte, h hash.Hash, sum []byte) []byte {
	h.Reset()
	h.Write(data)
	firstHash := h.Sum(sum)
	h.Reset()
	h.Write(firstHash)
	secondHash := h.Sum(firstHash[:0])
	return secondHash
}

func TestNetCheckAdressw(w io.Writer, pubKey []byte, sh256, rm hash.Hash, sum []byte) {
	temp := bytes.NewBuffer(make([]byte, 0, 25))
	temp.WriteByte(0x6f)
	RawAdressw(temp, pubKey, sh256, rm, sum[:0])
	check := doubleHash(temp.Bytes(), sh256, sum[:0])
	temp.Write(check[:4])
	io.Copy(w, temp)
}

func TestNetCheckAdress(pubKey []byte, sh256, rm hash.Hash, sum []byte) [25]byte {
	addr := [25]byte{}
	buf := bytes.NewBuffer(addr[:0])
	TestNetCheckAdressw(buf, pubKey, sh256, rm, sum)
	return addr
}
