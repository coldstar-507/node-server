package handlers

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"

	"fmt"
	"io"
	"log"

	"net/http"

	fg "github.com/coldstar-507/flatgen"
	"github.com/coldstar-507/node-server/internal/bsv"
	"github.com/coldstar-507/utils/utils"
	"go.mongodb.org/mongo-driver/bson"
	// "github.com/vmihailenco/msgpack/v5"
)

// const taal_api_key = "testnet_3860616b1cf1bb23110db44440f65899"

func PostTx(tx *bsv.Tx) error {
	rawTx := tx.Raw()
	rawTxHex := hex.EncodeToString(rawTx)
	log.Printf("ready tx\n%v", tx.Formatted())
	log.Printf("raw hex tx\n%v\n", rawTxHex)
	txPayload := map[string]any{"txhex": rawTxHex}
	rawPayload, err := json.Marshal(txPayload)
	if err != nil {
		return fmt.Errorf("postTx: err marshalling txPayload: %v", err)
	}
	payloadRdr := bytes.NewReader(rawPayload)

	const url string = "https://api.whatsonchain.com/v1/bsv/test/tx/raw"

	rsp, err := http.Post(url, "application/json", payloadRdr)
	if err != nil {
		return fmt.Errorf("postTx: error making broadcast request: %v", err)
	}

	if rsp.StatusCode != 200 {
		rbuf, _ := io.ReadAll(rsp.Body)
		return fmt.Errorf("postTx: error broadcasting tx: %v", string(rbuf))
	}

	var rjson map[string]any
	err = json.NewDecoder(rsp.Body).Decode(&rjson)
	if err != nil {
		return fmt.Errorf("postTx: error decoding response: %v", err)
	}

	status, ok := rjson["status"].(int)
	if !ok {
		return fmt.Errorf("postTx: error finding response status")
	}

	if status != 200 {
		title, detail := rjson["title"], rjson["detail"]
		return fmt.Errorf("%s: %d: %s", title, status, detail)
	} else if b, err := json.MarshalIndent(rjson, "", "    "); err != nil {
		log.Println("PostTx: error marshalling response with indent:", err)
		return nil
	} else {
		log.Println("PostTx:\n", string(b))
	}
	return nil

}

func ScanAreas(ctx context.Context, bq *fg.BoostQueryT) (map[uint16][]*bsv.User, int) {
	// limit of individual boosts
	curLim := bq.Lim
	// this is a map from chat places to list of users
	// we write the boosts to the user closest chatplace
	musers := make(map[uint16][]*bsv.User)
	// a boost request comes with a list of desired boost area selected by the client
	// we scan each area
	for _, a := range bq.Areas {
		usrs, newlim := bsv.ScanArea(a, bq, curLim)
		for place, us := range usrs {
			musers[place] = append(musers[place], us...)
		}
		curLim = newlim
		if curLim == 0 {
			break
		}
	}
	return musers, int(bq.Lim - curLim)
}

func SimpleQ(ctx context.Context, q *fg.BoostQueryT) (map[uint16][]*bsv.User, int) {
	// la, li := layer, curlim
	limit := q.Lim
	cursor, err := bsv.BoostQuery([]string{}, limit, q)
	if err != nil {
		log.Println("error making a query:", err)
		return nil, 0
		// continue
	}

	musers := make(map[uint16][]*bsv.User)
	for cursor.Next(ctx) {
		var usr bsv.User
		if err := bson.Unmarshal(cursor.Current, &usr); err != nil {
			log.Println("ScanArea: error unmarshalling bson:", err)
			break
		}
		if u, err := json.MarshalIndent(&usr, "", "    "); err != nil {
			log.Println("ScanArea: error pretty user:", err)
		} else {
			log.Println(string(u))
		}
		if len(usr.ChatPlaces) < 1 {
			log.Println("WARNING: ScanArea: user has no chat places")
			continue
		}
		usrCp := usr.ChatPlaces[0]
		musers[usrCp] = append(musers[usrCp], &usr)
		limit--
		if limit == 0 {
			break
		}
	}

	return musers, int(q.Lim - limit)
}

func writeTempMedia(r io.Reader, ch chan *os.File) {
	var temp *os.File
	temp_, err := os.CreateTemp("", "boost-media-*")
	if err != nil {
		ch <- nil
	}
	temp = temp_
	if _, err = io.Copy(temp, r); err != nil {
		ch <- nil
	}
	ch <- temp
}

func deleteTemp(temp *os.File) {
	if closeErr := temp.Close(); closeErr != nil {
		log.Printf("deleteTemp: file close err %s: %v", temp.Name(), closeErr)
	}
	if removeErr := os.Remove(temp.Name()); removeErr != nil {
		log.Printf("deleteTemp: file remove err %s: %v", temp.Name(), removeErr)
	}
}

func HandleBoostRequest(w http.ResponseWriter, r *http.Request) {
	ln := utils.Pln("HandleBoostRequest:")
	var (
		ctx    = context.Background()
		reqLen uint16
		req    []byte
	)
	utils.ReadBin(r.Body, &reqLen)
	req = make([]byte, reqLen)
	utils.ReadBin(r.Body, &req)
	boostReq := fg.GetRootAsBoostRequest(req, 0)
	var ch chan *os.File
	if boostReq.MediaLength() > 0 {
		ch = make(chan *os.File)
		go writeTempMedia(r.Body, ch)
	}

	bq := boostReq.Query(nil).UnPack()

	// boost comes with a partially filled tx
	// missing the outputs (each individual boost) and change
	tx := bsv.TxFromRdr(bytes.NewReader(boostReq.TxBytes()))
	ln("tx pre boost:", tx.Formatted())

	var musers map[uint16][]*bsv.User
	var nOuts int
	if len(bq.Areas) > 0 {
		musers, nOuts = ScanAreas(ctx, bq)
	} else {
		musers, nOuts = SimpleQ(ctx, bq)
	}
	// musers, nOuts := utils.If(len(bq.Areas) > 0, ScanAreas(ctx, bq), SimpleQ(ctx, bq))
	if nOuts == 0 {
		ln("Haven't found any people to boost")
		w.WriteHeader(501)
		return
	}

	// this fills the tx
	rdyTx := bsv.BoostScript(tx, boostReq, nOuts)

	var temp *os.File
	if ch != nil {
		temp = <-ch
		if temp == nil {
			w.WriteHeader(504)
			ln("error writing media to temp file")
		}
		defer deleteTemp(temp)
	}

	// we then broadcast the tx
	if err := PostTx(rdyTx); err != nil {
		ln("PostTex error", err)
		w.WriteHeader(502)
		return
	}

	// we then write the boosts on the chat servers

	bsv.WriteBoosts(musers, boostReq, bq.Interests, rdyTx.Txid(), temp)

	header := "Completed Boost"
	ntfBody := fmt.Sprintf("Found %v targets", nOuts)

	if _, err := bsv.SendNotification(header, ntfBody, string(boostReq.Token())); err != nil {
		ln("error pushing notification to booster:", err)
	}
}
