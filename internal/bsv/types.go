package bsv

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
)

// var curv elliptic.Curve = elliptic.

// Rules for encoding bsv transaction
// 4 byte int (uint32) -> LITTLE ENDIAN // versionNo, nLockTime
// 8 byte int (uint64) -> LITTLE ENDIAN // Satoshis
// VarInt              -> BIG ENDIAN

type Keys struct {
	priv ecdsa.PrivateKey
	pub  ecdsa.PublicKey
}

type Txin struct {
	txid       []uint8
	utxoIndex  uint32
	scriptLen  VarInt
	script     []uint8
	sequenceNo uint32
}

type Txout struct {
	sats      uint64
	scriptLen VarInt
	script    []uint8
}

type Tx struct {
	versionNo, nLockTime uint32
	nIns, nOuts          VarInt
	txouts               []*Txout
	txins                []*Txin
}

type VarInt struct {
	data []byte
	uint uint64
}

func txinFromRdr(rdr *bytes.Reader) *Txin {
	txidbuf := make([]byte, 32)
	var utxoi, seqno uint32

	rdr.Read(txidbuf)
	// binary.Read(rdr, binary.LittleEndian, txidbuf)
	binary.Read(rdr, binary.LittleEndian, &utxoi)
	scriptLen := varIntFromRdr(rdr)
	scriptBuf := make([]byte, scriptLen.uint)
	rdr.Read(scriptBuf)
	binary.Read(rdr, binary.LittleEndian, &seqno)

	return &Txin{
		txid:       txidbuf,
		utxoIndex:  utxoi,
		scriptLen:  scriptLen,
		script:     scriptBuf,
		sequenceNo: seqno,
	}

}

func (tin *Txin) raw() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, tin.txid)
	binary.Write(buf, binary.LittleEndian, tin.utxoIndex)
	buf.Write(tin.scriptLen.data)
	buf.Write(tin.script)
	binary.Write(buf, binary.LittleEndian, tin.sequenceNo)
	return buf.Bytes()
}

func txoutFromRdr(rdr *bytes.Reader) *Txout {
	var sats uint64
	binary.Read(rdr, binary.LittleEndian, &sats)
	scriptLen := varIntFromRdr(rdr)
	scriptBuf := make([]byte, scriptLen.uint)
	rdr.Read(scriptBuf)
	return &Txout{
		sats:      sats,
		scriptLen: scriptLen,
		script:    scriptBuf,
	}

}

func (tout *Txout) raw() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, tout.sats)
	buf.Write(tout.scriptLen.data)
	buf.Write(tout.script)
	return buf.Bytes()
}

func TxFromRdr(rdr *bytes.Reader) *Tx {
	var vNo, nLock uint32

	binary.Read(rdr, binary.LittleEndian, &vNo)
	nIns := varIntFromRdr(rdr)
	ins := make([]*Txin, 0, nIns.uint)
	for i := 0; i < int(nIns.uint); i++ {
		ins = append(ins, txinFromRdr(rdr))
	}
	nOuts := varIntFromRdr(rdr)
	outs := make([]*Txout, 0, nOuts.uint)
	for i := 0; i < int(nOuts.uint); i++ {
		outs = append(outs, txoutFromRdr(rdr))
	}
	binary.Read(rdr, binary.LittleEndian, &nLock)

	return &Tx{
		versionNo: vNo,
		nIns:      nIns,
		txins:     ins,
		nOuts:     nOuts,
		txouts:    outs,
		nLockTime: nLock,
	}
}

func (t *Tx) Raw() []byte {
	buf := new(bytes.Buffer)
	inFold, outFold := []byte{}, []byte{}
	for _, tin := range t.txins {
		inFold = append(inFold, tin.raw()...)
	}
	for _, tout := range t.txouts {
		outFold = append(outFold, tout.raw()...)
	}

	binary.Write(buf, binary.LittleEndian, t.versionNo)
	buf.Write(t.nIns.data)
	buf.Write(inFold)
	buf.Write(t.nOuts.data)
	buf.Write(outFold)
	binary.Write(buf, binary.LittleEndian, t.nLockTime)
	return buf.Bytes()
}

func Txid(rawTx []byte) []byte {
	h := sha256.New()
	h.Write(rawTx)
	k := h.Sum(nil)
	h.Reset()
	h.Write(k)
	return h.Sum(nil)
}

func (t *Tx) Txid() []byte {
	return Txid(t.Raw())
}

func (t *Tx) TxidHex() string {
	return hex.EncodeToString(t.Txid())
}

func (t *Tx) TxidHexR() string {
	txid := t.Txid()
	slices.Reverse(txid)
	return hex.EncodeToString(txid)
}

func varIntFromRdr(rdr *bytes.Reader) VarInt {
	var uint uint64
	firstByte, _ := rdr.ReadByte()
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, firstByte)

	if firstByte == 0xFF {
		var d uint64
		binary.Read(rdr, binary.BigEndian, &d)
		binary.Write(buf, binary.BigEndian, d)
		uint = d
	} else if firstByte == 0xFE {
		var d uint32
		binary.Read(rdr, binary.BigEndian, &d)
		binary.Write(buf, binary.BigEndian, d)
		uint = uint64(d)
	} else if firstByte == 0xFD {
		var d uint16
		binary.Read(rdr, binary.BigEndian, &d)
		binary.Write(buf, binary.BigEndian, d)
		uint = uint64(d)
	} else {
		uint = uint64(firstByte)
	}

	return VarInt{uint: uint, data: buf.Bytes()}
}

func makeVarInt(n int) VarInt {
	buf := new(bytes.Buffer)
	if n >= 0 && n <= 252 {
		binary.Write(buf, binary.BigEndian, uint8(n))
	} else if n >= 253 && n <= 65535 {
		binary.Write(buf, binary.BigEndian, uint8(0xFD))
		binary.Write(buf, binary.BigEndian, uint8(n))
	} else if n >= 65536 && n <= 4294967295 {
		binary.Write(buf, binary.BigEndian, uint8(0xFE))
		binary.Write(buf, binary.BigEndian, uint32(n))
	} else {
		binary.Write(buf, binary.BigEndian, uint8(0xFF))
		binary.Write(buf, binary.BigEndian, uint64(n))
	}
	return VarInt{data: buf.Bytes(), uint: uint64(n)}
}

func (tin *Txin) Formatted() string {
	txid, script := hex.EncodeToString(tin.txid), hex.EncodeToString(tin.script)
	return fmt.Sprintf(`
==TXIN==
utxoTxid : %v
utxoIndex: %v
scriptLen: %v
script   : %v
seqNo    : %v
========
`, txid, tin.utxoIndex, tin.scriptLen.uint, script, tin.sequenceNo)
}

func (tout *Txout) Formatted() string {
	script := hex.EncodeToString(tout.script)
	return fmt.Sprintf(`
==TXOUT==
sats     : %v
scriptLen: %v
script   : %v
=========
`, tout.sats, tout.scriptLen.uint, script)
}

func (tx *Tx) Formatted() string {
	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf(`
==TX==
versionNo: %v
inCount  : %v
`, tx.versionNo, tx.nIns.uint))
	for _, tin := range tx.txins {
		buf.WriteString(tin.Formatted())
	}
	buf.WriteString(fmt.Sprintf("outCount=%v\n", tx.nOuts.uint))
	for _, tout := range tx.txouts {
		buf.WriteString(tout.Formatted())
	}
	buf.WriteString(fmt.Sprintf("nLockTime=%v\n===========\n", tx.nLockTime))
	return buf.String()
}
