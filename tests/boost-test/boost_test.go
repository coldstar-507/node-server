package test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/coldstar-507/flatgen"
	"github.com/coldstar-507/node-server/internal/bsv"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"github.com/coldstar-507/router/router_utils"
	"github.com/coldstar-507/utils/utils"

	"go.mongodb.org/mongo-driver/bson"
)

func TestMain(m *testing.M) {
	db.InitMongo()
	defer db.ShutdownMongo()
	mr := router_utils.FetchMetaRouter()
	log.Println("TestMain(...): metaRouter:\n", utils.SprettyPrint(mr))
	router_utils.SetMetaRouter(mr)
	code := m.Run()
	os.Exit(code)
}

func TestBoost(t *testing.T) {
	ln, f := utils.Pln("TestBoost:"), utils.Pf("TestBoost: ")
	ctx := context.Background()

	b, err := os.ReadFile("flatNoMedia")
	if err != nil {
		t.Error(err)
	}

	raw := make([]byte, len(b)/2)
	if _, err := hex.Decode(raw, b); err != nil {
		t.Error(err)
	}

	fbr := flatgen.GetRootAsBoostRequest(raw, 0)
	bq := fbr.Query(nil).UnPack()

	tx := bsv.TxFromRdr(bytes.NewReader(fbr.TxBytes()))
	f("hex partial tx:\n%X\n", fbr.TxBytes())
	ln("formatted tx:\n", tx.Formatted())

	var musers map[uint16][]*bsv.User
	var nOuts int
	if len(bq.Areas) > 0 {
		musers, nOuts = handlers.ScanAreas(ctx, bq)
	} else {
		musers, nOuts = handlers.SimpleQ(ctx, bq)
	}

	i, _ := db.Nodes.CountDocuments(ctx, bson.D{})
	ln(i, "nodes")

	ln("found", nOuts, "users")
	if nOuts == 0 {
		ln("no users found for boost")
	}

	for _, user := range musers {
		p, err := json.MarshalIndent(&user, "", "    ")
		if err != nil {
			log.Println(string(p))
		}
	}

	rdyTx := bsv.BoostScript(tx, fbr, nOuts)
	f("readyTx id:\n%x\n", rdyTx.Txid())
	f("readyTx:\n%x\n", rdyTx.Raw())
	f("formatted readyTx:\n%s\n", rdyTx.Formatted())
	bsv.WriteBoosts(musers, fbr, bq.Interests, rdyTx.Txid(), nil)

}
