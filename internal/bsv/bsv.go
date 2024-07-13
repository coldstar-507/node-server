package bsv

import (
	"bytes"
	"crypto/sha1"
	"hash"
	"math"

	"encoding/binary"
)

func simpleBoostHashPuzzle(s1 []byte, i uint32, buf *bytes.Buffer, h hash.Hash) []byte {
	shp := make([]byte, 23)
	buf.Write(s1)
	binary.Write(buf, binary.BigEndian, i)
	h.Write(buf.Bytes())
	s2 := h.Sum(nil)
	h.Reset()
	h.Write(s2)
	s3 := h.Sum(nil)
	h.Reset()
	buf.Reset()
	buf.WriteByte(op_sha1)
	buf.Write(op_push_data(s3))
	buf.WriteByte(op_equal)
	copy(shp, buf.Bytes())
	buf.Reset()
	return shp
}

func p2pkh(pkh []byte, buf *bytes.Buffer) []byte {
	sc := make([]byte, 25)
	buf.WriteByte(op_dup)
	buf.WriteByte(op_hash160)
	buf.Write(op_push_data(pkh))
	buf.WriteByte(op_equal_verify)
	buf.WriteByte(op_check_sig)
	copy(sc, buf.Bytes())
	buf.Reset()
	return sc
}

func BoostScript(t *Tx, s1 []byte, nout int, pph int, inSats int, addr []byte) *Tx {
	// const bytes_per_sat float64 = 20
	const bytes_per_sat float64 = 1000 // fees are len(tx.raw()) / 1000
	const shp_out_size = 32            // 3 OPS, 20 data_bytes, 8 bytes for sats, 1 byte for len
	const p2pkh_out_size = 34          // 5 OPS, 20 data_bytes, 8 bytes for sats, 1 byte for len

	buf, h := new(bytes.Buffer), sha1.New()
	outs := make([]*Txout, 0, nout+1)

	for i := 0; i < nout; i++ {
		shp := simpleBoostHashPuzzle(s1, uint32(i), buf, h)
		tout := &Txout{
			sats:      uint64(pph),
			scriptLen: makeVarInt(len(shp)),
			script:    shp,
		}
		outs = append(outs, tout)
	}

	vout := makeVarInt(nout + 1)                                          // nout + change
	outRelSize := (shp_out_size * nout) + p2pkh_out_size + len(vout.data) // size relating outs
	txSize := len(t.Raw()) - 1 + outRelSize                               // -1 to remove varInt(0) vout

	fees := int(math.Ceil(float64(txSize) / bytes_per_sat))
	boostSats := pph * nout
	change := inSats - (boostSats + fees)

	changeScript := p2pkh(addr, buf)
	changeOut := &Txout{
		sats:      uint64(change),
		scriptLen: makeVarInt(len(changeScript)),
		script:    changeScript,
	}

	outs = append(outs, changeOut)

	// s1 = hash(booster_secret + nonce) // on server alongside boost
	// s2 = hash(s1 + uint32(i_out))
	// s3 = hash(s2) // this is what we put in the output
	// full script -> OP_PUSH(s2) | OP_HASH* OP_PUSH(s3) OP_EQUAL
	// 68 bytes txs with sha256 hashes -> 1M outs mean 68 mb
	// 44 bytes txs with sha1 hashes -> 1M outs mean 44 mb
	// ouputs being 23 bytes for script, 8 bytes for sats -> 31 bytes
	// inputs depends if they are p2pkh or
	// maximun is 2^32 outs which would be 44 * ~4billion // which would be insane

	t.txouts = outs
	t.nOuts = vout
	return t

}
