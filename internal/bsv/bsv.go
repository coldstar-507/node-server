package bsv

import (
	"bytes"
	"hash"
	"io"

	// "crypto/sha1"
	"crypto/sha256"
	"math"

	"github.com/coldstar-507/flatgen"
	"github.com/coldstar-507/utils2"
	"golang.org/x/crypto/ripemd160"
)

func shpw(w io.Writer, s1 []byte, i uint32, h hash.Hash) {
	h.Reset()
	utils2.WriteBin(h, s1, i)
	s2 := h.Sum(nil)
	utils2.WriteBin(w, op_sha1, op_push_data(s2), op_equal)
}

func makeShp(s1 []byte, i uint32, h hash.Hash) [23]byte {
	script := [23]byte{}
	buf := bytes.NewBuffer(script[:0])
	shpw(buf, s1, i, h)
	return script
}

func p2pkhw(w io.Writer, addr []byte) {
	utils2.WriteBin(w,
		op_dup,
		op_hash160,
		op_push_data(addr),
		op_equal_verify,
		op_check_sig)
}

func p2pkh(addr []byte) [25]byte {
	script := [25]byte{}
	buf := bytes.NewBuffer(script[:0])
	p2pkhw(buf, addr)
	return script
}

// func BoostScript(t *Tx, s1 []byte, nout, pph, inSats int, addr []byte) *Tx {
// 	// const bytes_per_sat float64 = 20
// 	// fees are len(tx.raw()) / 1000
// 	const bytes_per_sat float64 = 1000
// 	const shp_out_size = 32   // 3 OPS, 20 data_bytes, 8 bytes for sats, 1 byte for len
// 	const p2pkh_out_size = 34 // 5 OPS, 20 data_bytes, 8 bytes for sats, 1 byte for len

// 	buf, h := new(bytes.Buffer), sha1.New()
// 	outs := make([]*Txout, 0, nout+1)

// 	for i := 0; i < nout; i++ {
// 		shp := simpleBoostHashPuzzle(s1, uint32(i), buf, h)
// 		tout := &Txout{
// 			sats:      uint64(pph),
// 			scriptLen: makeVarInt(len(shp)),
// 			script:    shp,
// 		}
// 		outs = append(outs, tout)
// 	}
// 	// nout + change
// 	vout := makeVarInt(nout + 1)
// 	// size relating outs
// 	outRelSize := (shp_out_size * nout) + p2pkh_out_size + len(vout.data)
// 	// -1 to remove varInt(0) vout
// 	txSize := len(t.Raw()) - 1 + outRelSize

// 	fees := int(math.Ceil(float64(txSize) / bytes_per_sat))
// 	boostSats := pph * nout
// 	change := inSats - (boostSats + fees)

// 	changeScript := p2pkh(addr, buf)
// 	changeOut := &Txout{
// 		sats:      uint64(change),
// 		scriptLen: makeVarInt(len(changeScript)),
// 		script:    changeScript,
// 	}

// 	outs = append(outs, changeOut)

// 	// s1 = hash(booster_secret + nonce) // on server alongside boost
// 	// s2 = hash(s1 + uint32(i_out))
// 	// s3 = hash(s2) // this is what we put in the output
// 	// full script -> OP_PUSH(s2) | OP_HASH* OP_PUSH(s3) OP_EQUAL
// 	// 68 bytes txs with sha256 hashes -> 1M outs mean 68 mb
// 	// 44 bytes txs with sha1 hashes -> 1M outs mean 44 mb
// 	// ouputs being 23 bytes for script, 8 bytes for sats -> 31 bytes
// 	// inputs depends if they are p2pkh or
// 	// maximun is 2^32 outs which would be 44 * ~4billion // which would be insane

// 	t.txouts = outs
// 	t.nOuts = vout
// 	return t

// }

// we always require having enough sats for a change output
// this simplifies the function greatly
func BoostScript(t *Tx, br *flatgen.BoostRequest, nout int) *Tx {
	// fees are len(tx.raw()) / 1000
	const bytes_per_sat float64 = 1000.0
	// const shp_out_size = 32   // 3 OPS, 20 data_bytes, 8 bytes for sats, 1 byte for len
	const p2pkh_out_size = 34 // 5 OPS, 20 data_bytes, 8 bytes for sats, 1 byte for len

	// full size is:
	// 4 + 4 + len(nIns.data) + sum(tin.Size()) + len(nOuts.data) + sum(tou.Size())
	var txSize = 4 + 4 + len(t.nIns.data)
	for _, tin := range t.txins {
		txSize += tin.Size()
	}
	vnOut := makeVarInt(nout + 1)
	txSize += len(vnOut.data)
	txSize += p2pkh_out_size * (nout + 1)

	fees := int(math.Ceil(float64(txSize) / bytes_per_sat))
	boostSats := int(br.PricePerHead()) * nout
	change := int(br.InputSats()) - (boostSats + fees)
	utils2.Assert(change > 0, "BoostScript: change=%d must be > 0", change)
	changeAddr := br.ChangeAddressBytes()
	utils2.Assert(len(changeAddr) == 20, "BoostScript: invalid address: %x", changeAddr)

	outs := make([]*Txout, 0, nout+1)
	changeScript := p2pkh(changeAddr)
	changeOut := &Txout{
		sats:      uint64(change),
		scriptLen: makeVarInt(len(changeScript)),
		script:    changeScript[:],
	}
	outs = append(outs, changeOut)

	sh256, rmd160, sum := sha256.New(), ripemd160.New(), [32]byte{}
	for i := uint32(1); i < uint32(nout)+1; i++ {
		secret := MakeSecret(br.S1Bytes(), i, sh256)
		keys := ProjectKeys.Derive(secret)
		addr := MakeRawAdress(keys.Pub.SerializeCompressed(), sh256, rmd160, sum[:0])
		script := p2pkh(addr[:])
		tout := &Txout{
			sats:      uint64(br.PricePerHead()),
			scriptLen: makeVarInt(len(script)),
			script:    script[:],
		}
		outs = append(outs, tout)
	}

	t.txouts = outs
	t.nOuts = vnOut
	return t

}
