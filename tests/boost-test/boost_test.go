package test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/coldstar-507/node-server/internal/bsv"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMain(m *testing.M) {
	db.InitMongo()
	defer db.ShutdownMongo()
	code := m.Run()
	os.Exit(code)
}

func TestBoost(t *testing.T) {
	ctx := context.Background()

	var br bsv.BoostTest
	if err := json.Unmarshal([]byte(brjson), &br); err != nil {
		t.Error("TestBoost err: unmarshalling brjson:", err)
	}

	rawTx, err := hex.DecodeString(br.PartialTx)
	if err != nil {
		t.Error("TestBoost err: decoding partialTx:", err)
	}

	tx := bsv.TxFromRdr(bytes.NewReader(rawTx))
	log.Println("TestBoost: hex partial tx:\n", br.PartialTx)
	log.Println("TestBoost: formatted tx:\n", tx.Formatted())

	musers, nOuts := handlers.ScanAreas(ctx,
		br.Genders, br.Interests, br.Areas, br.Limit, br.MinAge, br.MaxAge)

	i, _ := db.Nodes.CountDocuments(ctx, bson.D{})
	log.Println(i, "nodes")

	log.Println("TestBoost: found", nOuts, "users")
	if nOuts == 0 {
		t.Error("TestBoost: no users found for boost")
	}

	for _, user := range musers {
		p, err := json.MarshalIndent(&user, "", "    ")
		if err != nil {
			log.Println(string(p))
		}
	}

	s1, err := hex.DecodeString(br.S1)
	if err != nil {
		t.Error("TestBoost: error decoding s1:", err)
	}

	changeAddr, err := hex.DecodeString(br.ChangeAddress)
	if err != nil {
		t.Error("TestBoost: error decoding changeAdress:", err)
	}

	rdyTx := bsv.BoostScript(tx, s1, nOuts, br.PricePerHead, br.InputSats, changeAddr)
	// 7b5e7c0e91cc886b9d3809ed3006d0b8b339be2aa3d96007eb79b8c621840cbc
	log.Println("TestBoost: readyTx id:\n", rdyTx.TxidHexR())
	log.Println("TestBoost: readyTx:\n", hex.EncodeToString(rdyTx.Raw()))
	log.Println("TestBoost: formatted readyTx:\n", rdyTx.Formatted())

	boostMsg, err := hex.DecodeString(br.BoostMessage)
	if err != nil {
		t.Error("TestBoost: error decoding boost message:", err)
	}

	bsv.WriteBoosts(musers, br.Interests, boostMsg, nil)

	// not posting that tx yet, want to make sure WriteBoosts works
	// if err := handlers.PostTx(rdyTx); err != nil {
	// 	t.Error("TestBoost:", err)
	// }

}

var brjson = `
 {
  "token": "cC6CREDRSpuaah1Jvr7YIg:APA91bHpubf6M4nFQ_FAPc_LO2-gq5tXjn0gewoFa-uLGK56aHSQ0zlHMMKoZKfozXn7tRos8VgLxrlg7A8EbxYwNKV3fbl3mTaJxOjgBE14qa5p760rpjFfgTJeZK7EQEUAVGemljGT",
  "senderId": "0500000192a125f715ec90b84f",
  "deviceId": 338937927,
  "limit": 5000,
  "maxAge": 200,
  "minAge": 18,
  "pph": 10,
  "genders": [
    "male"
  ],
  "interests": [
    "coffee"
  ],
  "s1": "da502a2d3cd68ef35932002c2904bbbb2c32f947",
  "changeAddr": "e7a9ea757be4845501b901e71e0bf1f0f0dc700d",
  "inputSats": 63777,
  "tx": "0100000004e96f31b42d47c498dc3c66d0b8000698131e6389503a71a0d3f4eaf7febd5bd5050000006a473044022068a431b123368d6976a174a0bac8e46a30b9ee928807501bd6ab17ab722abd4d022035b8168fba0a822f2158c22daef7de6dcc6c8b17b89d5732dd6caf587a9d8056422102c906b9c62a165cb303c4b3bf0b889cedba34a24bbbf41f136064c4ee7223f45affffffffe96f31b42d47c498dc3c66d0b8000698131e6389503a71a0d3f4eaf7febd5bd5070000006a473044022051749c12d118ad79433befcdc75f2d8133657689f33c852b62d745a0974774fb0220453043bc34db255bbc680e36ab763cfa4bb9f0639b67a250e39b864793c32c53422102c906b9c62a165cb303c4b3bf0b889cedba34a24bbbf41f136064c4ee7223f45affffffff98a47423b1c665afdb66f1f15ce1e074500804d60f47a03070976d5241bc3f4f040000006a47304402202adb081dbf95ca6daa39b26b08cf2f9319cb50356b0f04f6b003b8c77bdab8db022068c75c69222d77d5d5c4ef671dd83cbe96a0b97e0778183c1d77ef8cb79fcead4221035bbd1db635265d3c6f7682e51adcc9f1501f9b6a66dbafbc950f0c12c2ad75d5ffffffff98a47423b1c665afdb66f1f15ce1e074500804d60f47a03070976d5241bc3f4f070000006b483045022100abacc5fc62c01d9b06fe24f3e5e0ee11e541ff86721b810496f8acdf634900a802203bcb08e9eec4f9f55c8464fd349a490de4beed8e12cc186facc6014ac4080b6f4221035bbd1db635265d3c6f7682e51adcc9f1501f9b6a66dbafbc950f0c12c2ad75d5ffffffff0000000000",
  "areas": [
    {
      "center": {
        "lat": 48.84113330035002,
        "lon": -67.52731264576636
      },
      "perim": [
        {
          "lat": 48.82223178594845,
          "lon": -67.52731264576636,
          "refDist": 2.101752507353629
        },
        {
          "lat": 48.827768649754134,
          "lon": -67.54761635429574,
          "refDist": 2.1016363833208405
        },
        {
          "lat": 48.84113330035002,
          "lon": -67.55602642573508,
          "refDist": 2.101356052217394
        },
        {
          "lat": 48.85449438575701,
          "lon": -67.54761635429574,
          "refDist": 2.101075745335871
        },
        {
          "lat": 48.86002768437386,
          "lon": -67.52731264576636,
          "refDist": 2.1009596455247928
        },
        {
          "lat": 48.85449438575701,
          "lon": -67.50700893723698,
          "refDist": 2.101075745335871
        },
        {
          "lat": 48.84113330035002,
          "lon": -67.49859886579766,
          "refDist": 2.101356052215314
        },
        {
          "lat": 48.827768649754134,
          "lon": -67.50700893723698,
          "refDist": 2.1016363833208405
        }
      ]
    }
  ],
  "boostMessage": "38000000000000000000000000002a002c0028002700000020001c000000000000001000000000000c0008000400000000000000000000002a000000280000004c0000004c0000000bedbbe3920100000000000064000000700000000000000098000000210000004e6963652e20416e6f746865722075676c79207069656365206f6620736869742e0000000000000001000000040000001a000000303530303030303139323737613031313364376134356666323100000800000063616c616d697479000000001a0000003035303030303031393261313235663731356563393062383466000000000e001a00190010000c00080007000e0000000000000720000000b6cd6e901259bae39201000000000e001e001d0018001400080006000e0000000000ffffffffffffffffff0000000000140000003000000000010a0016000800040000000a0000000000000000000000000000000000000000000a0010000800040000000a000000000000000000000000000000"
}
`
