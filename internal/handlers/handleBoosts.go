package handlers

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"

	"fmt"
	"io"
	"log"
	"math"

	"net/http"

	"github.com/coldstar-507/node-server/internal/bsv"
	"github.com/vmihailenco/msgpack/v5"
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

func ScanAreas(ctx context.Context, genders, interests []string,
	areas []*bsv.Area, limit, minAge, maxAge int) (map[uint16][]*bsv.User, int) {
	// limit of individual boosts
	curLim := limit
	// this is a map from chat places to list of users
	// we write the boosts to the user closest chatplace
	musers := make(map[uint16][]*bsv.User)
	// a boost request comes with a list of desired boost area selected by the client
	// we scan each area
	for _, a := range areas {
		usrs, newlim := bsv.ScanArea(ctx, a, genders, interests, minAge, maxAge, curLim)
		for place, us := range usrs {
			musers[place] = append(musers[place], us...)
		}
		curLim = newlim
		if curLim == 0 {
			break
		}
	}
	return musers, limit - curLim
}

func HandleBoostRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	// this httpReq begins with a BoostRequest
	var br bsv.BoostRequest
	if err := msgpack.NewDecoder(r.Body).Decode(&br); err != nil {
		w.WriteHeader(500)
		log.Println("HandleBoostRequest error",
			"decoding boostRequest:", err)
		return
	}

	// price attached (in sats) to each individual boost
	if br.PricePerHead > math.MaxUint32 {
		w.WriteHeader(500)
		log.Println("HeandlerBoostRequest error",
			"PricePerHead exceeds maximum amount of 42 bsv:",
			br.PricePerHead)
		return
	}

	// boost comes with a partially filled tx
	// missing the outputs (each individual boost) and change
	tx := bsv.TxFromRdr(bytes.NewReader(br.PartialTx))
	log.Println("tx pre boost:", tx.Formatted())

	musers, nOuts := ScanAreas(ctx,
		br.Genders, br.Interests, br.Areas, br.MinAge, br.MaxAge, br.Limit)

	if nOuts == 0 {
		log.Println("HandleBoostRequest: Haven't found any people to boost")
		w.WriteHeader(500)
		return
	}

	// this fills the tx
	rdyTx := bsv.BoostScript(tx, br.S1, nOuts,
		br.PricePerHead, br.InputSats, br.ChangeAddress)

	// we then broadcast the tx
	if err := PostTx(rdyTx); err != nil {
		log.Println("HandleBoostRequest:", err)
		w.WriteHeader(500)
		return
	}

	// we then write the boosts on the chat servers
	bsv.WriteBoosts(musers, br.Interests, br.BoostMessage, br.FullMedia)

	header := "Completed Boost"
	body := fmt.Sprintf("Found %v targets", nOuts)

	if _, err := bsv.SendNotification(header, body, br.Token); err != nil {
		log.Println("error pushing notification to booster:", err)
	}
}
