package bsv

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/coldstar-507/utils2"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

var curv = secp256k1.S256()

// Rules for encoding bsv transaction
// 4 byte int (uint32) -> LITTLE ENDIAN // versionNo, nLockTime
// 8 byte int (uint64) -> LITTLE ENDIAN // Satoshis
// VarInt              -> BIG ENDIAN

var ProjectKeys *Keys = loadKeysFromEnv()

func loadKeysFromEnv() *Keys {
	fn, ok := os.LookupEnv("BOOSTS_KEY_FILE")
	if !ok {
		panic("ENV missing variable: BOOSTS_KEY_FILE")
	}
	f, err := os.Open(fn)
	utils2.Must(err)
	kbuf := make([]byte, 256)
	n, err := f.Read(kbuf)
	utils2.Must(err)
	k, err := KeysFromJson(kbuf[:n])
	utils2.Must(err)
	return k
}

type Keys struct {
	Prv *secp256k1.PrivateKey
	Pub *secp256k1.PublicKey
	Cc  []byte
}

func NewKeys() (*Keys, error) {
	cc := utils2.RandomBytes(32)
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	return &Keys{Cc: cc, Prv: priv, Pub: priv.PubKey()}, nil
}

func (k *Keys) ToJson() map[string]any {
	m := map[string]any{
		"cc":  hex.EncodeToString(k.Cc),
		"pub": hex.EncodeToString(k.Pub.SerializeCompressed()),
	}
	if k.Prv != nil {
		m["prv"] = hex.EncodeToString(k.Prv.Serialize())
	}
	return m
}

func (k *Keys) ToJsonEncoded(pretty bool) []byte {
	if pretty {
		b, err := json.MarshalIndent(k.ToJson(), "", "    ")
		utils2.Must(err)
		return b
	} else {
		b, err := json.Marshal(k.ToJson())
		utils2.Must(err)
		return b
	}
}

func KeysFromJson(j []byte) (*Keys, error) {
	var m map[string]any
	if err := json.Unmarshal(j, &m); err != nil {
		return nil, err
	}
	ccHex, ok0 := m["cc"].(string)
	pubHex, ok1 := m["pub"].(string)
	if !ok0 || !ok1 {
		return nil, fmt.Errorf("ccHex or pubHex isn't string")
	}

	cc, err0 := hex.DecodeString(ccHex)
	rawPub, err1 := hex.DecodeString(pubHex)
	if err := errors.Join(err0, err1); err != nil {
		return nil, err
	}

	pub, err := secp256k1.ParsePubKey(rawPub)
	if err != nil {
		return nil, err
	}

	if hprv := m["prv"]; hprv != nil {
		prvHex, ok2 := hprv.(string)
		if !ok2 {
			return nil, fmt.Errorf("prvHex isn't string")
		}

		rawPrv, err := hex.DecodeString(prvHex)
		if err != nil {
			return nil, err
		}

		prv := secp256k1.PrivKeyFromBytes(rawPrv)
		return &Keys{Cc: cc, Pub: pub, Prv: prv}, nil
	} else {
		return &Keys{Cc: cc, Pub: pub}, nil
	}
}

func (k *Keys) IsNeutered() bool {
	return k.Prv == nil
}

func (k *Keys) Derive(secret []byte) *Keys {
	hm := hmac.New(sha512.New, k.Cc)
	ecpub := k.Pub.ToECDSA()
	hm.Write(ecpub.X.Bytes())
	hm.Write(ecpub.Y.Bytes())
	hm.Write(secret)
	out := hm.Sum(nil)
	left, right := out[:32], out[32:64]
	bigr := new(secp256k1.ModNScalar)
	bigr.SetByteSlice(right)
	var rJp, pubJp, newPubJp secp256k1.JacobianPoint
	secp256k1.ScalarBaseMultNonConst(bigr, &rJp)
	k.Pub.AsJacobian(&pubJp)
	secp256k1.AddNonConst(&rJp, &pubJp, &newPubJp)

	newPubJp.ToAffine()
	newPub := secp256k1.NewPublicKey(&newPubJp.X, &newPubJp.Y)

	if k.IsNeutered() {
		return &Keys{Pub: newPub, Cc: left}
	} else {
		newSca := k.Prv.Key.Add(bigr)
		newPriv := secp256k1.NewPrivateKey(newSca)
		return &Keys{Pub: newPub, Prv: newPriv, Cc: left}
	}
}

type Txin struct {
	txid       []uint8
	utxoIndex  uint32
	scriptLen  VarInt
	script     []uint8
	sequenceNo uint32
}

func (tin *Txin) Size() int {
	return 32 + 4 + len(tin.scriptLen.data) + len(tin.script) + 4
}

type Txout struct {
	sats      uint64
	scriptLen VarInt
	script    []uint8
}

func (tou *Txout) Size() int {
	return 8 + len(tou.scriptLen.data) + len(tou.script)
}

type Tx struct {
	versionNo, nLockTime uint32
	nIns, nOuts          VarInt
	txouts               []*Txout
	txins                []*Txin
}

func (tx *Tx) Size() int {
	var sum = 4 + 4 + len(tx.nIns.data) + len(tx.nOuts.data)
	for _, tou := range tx.txouts {
		sum += tou.Size()
	}
	for _, tin := range tx.txins {
		sum += tin.Size()
	}
	return sum
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

	switch firstByte {
	case 0xFF:
		var d uint64
		binary.Read(rdr, binary.BigEndian, &d)
		binary.Write(buf, binary.BigEndian, d)
		uint = d
	case 0xFE:
		var d uint32
		binary.Read(rdr, binary.BigEndian, &d)
		binary.Write(buf, binary.BigEndian, d)
		uint = uint64(d)
	case 0xFD:
		var d uint16
		binary.Read(rdr, binary.BigEndian, &d)
		binary.Write(buf, binary.BigEndian, d)
		uint = uint64(d)
	default:
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
	fmt.Fprintf(buf, `
==TX==
versionNo: %v
inCount  : %v
`, tx.versionNo, tx.nIns.uint)

	for _, tin := range tx.txins {
		buf.WriteString(tin.Formatted())
	}
	fmt.Fprintf(buf, "outCOunt=%v\n", tx.nOuts.uint)
	for _, tout := range tx.txouts {
		buf.WriteString(tout.Formatted())
	}
	fmt.Fprintf(buf, "nLockTime=%v\n===========\n", tx.nLockTime)
	return buf.String()
}
