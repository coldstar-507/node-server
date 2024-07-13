package bsv

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"

	"firebase.google.com/go/v4/messaging"
	"github.com/coldstar-507/flatgen"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/utils"
	"github.com/mmcloughlin/geohash"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const taal_api_key = "testnet_3860616b1cf1bb23110db44440f65899"

type latlon struct {
	Lat     float64 `msgpack:"lat"`
	Lon     float64 `msgpack:"lon"`
	RefDist float64 `msgpack:"refDist"`
}

const precision = 4

func sendNotification(header, body, token string) (string, error) {
	return db.Messager.Send(context.Background(), &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: header,
			Body:  body,
		}})
}

func geoDist(ll1, ll2 latlon) float64 {
	lat1, lon1, lat2, lon2 := ll1.Lat, ll1.Lon, ll2.Lat, ll2.Lon
	R := 6371.0                   // Radius of the earth in km
	dLat := degToRad(lat2 - lat1) // degToRad below
	dLon := degToRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degToRad(lat1))*math.Cos(degToRad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := R * c // Distance in km
	return d
}

func geoDist_(lat1, lon1, lat2, lon2 float64) float64 {
	R := 6371.0                   // Radius of the earth in km
	dLat := degToRad(lat2 - lat1) // degToRad below
	dLon := degToRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degToRad(lat1))*math.Cos(degToRad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := R * c // Distance in km
	return d
}

func geoBearing(ll1, ll2 latlon) float64 {
	lat1, lon1, lat2, lon2 := ll1.Lat, ll1.Lon, ll2.Lat, ll2.Lon
	// Convert degrees to radians
	lat1Rad := degToRad(lat1)
	lon1Rad := degToRad(lon1)
	lat2Rad := degToRad(lat2)
	lon2Rad := degToRad(lon2)

	// Calculate angle using spherical law of cosines
	angle := math.Acos(math.Sin(lat1Rad)*math.Sin(lat2Rad) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Cos(lon2Rad-lon1Rad))

	// Convert angle from radians to degrees
	angle = radToDeg(angle)

	return angle
}

func degToRad(deg float64) float64 {
	return deg * (math.Pi / 180)
}

func radToDeg(rad float64) float64 {
	return rad * 180.0 / math.Pi
}

func closest(l []latlon, p latlon) latlon {
	ix, smallDist := int(0), math.MaxFloat64
	for i, x := range l {
		dist := geoDist(x, p)
		if dist < smallDist {
			smallDist = dist
			ix = i
		}
	}
	return l[ix]
}

func closest2(l []latlon, p latlon) []latlon {
	ix, smallDist := int(0), math.MaxFloat64
	ix2, smallDist2 := int(0), math.MaxFloat64
	for i, x := range l {
		dist := geoDist(x, p)
		if dist < smallDist {
			smallDist = dist
			ix = i
		}
		if dist > smallDist && dist < smallDist2 {
			smallDist2 = dist
			ix2 = i
		}
	}
	return []latlon{l[ix], l[ix2]}
}

func validHash3(a area, hash string) bool {
	box := geohash.BoundingBox(hash)
	bclat, bclon := box.Center()
	boxCenter := latlon{Lat: bclat, Lon: bclon}
	bounds := []latlon{
		{Lat: box.MaxLat, Lon: box.MaxLng},
		{Lat: box.MaxLat, Lon: box.MinLng},
		{Lat: box.MinLat, Lon: box.MaxLng},
		{Lat: box.MinLat, Lon: box.MinLng},
	}

	twoClosest := closest2(a.Perim, boxCenter)
	for _, b := range bounds {
		valid := utils.Any(twoClosest, func(ll latlon) bool {
			return geoDist(b, a.Center) < ll.RefDist
		})
		if valid {
			return true
		}
	}
	return false
}

func calcLayers2(a area) [][]string {
	clat, clon := a.Center.Lat, a.Center.Lon
	centerHash := geohash.EncodeWithPrecision(clat, clon, precision)
	layers := [][]string{{centerHash}}
	flat := []string{centerHash}

	var getLayers func([]string) [][]string
	getLayers = func(l []string) [][]string {
		curLayer := make([]string, 0)
		for _, e := range l {
			nbs := geohash.Neighbors(e)
			for _, nb := range nbs {
				if !utils.Contains(nb, flat) && validHash3(a, nb) {
					flat = append(flat, nb)
					curLayer = append(curLayer, nb)
				}
			}
		}

		nValid := len(curLayer)
		if nValid > 0 {
			layers = append(layers, curLayer)
		} else {
			return layers
		}

		if nValid > len(l) {
			return getLayers(curLayer)
		} else {
			return layers
		}

	}

	getLayers(layers[0])

	// required for firestore...
	// mongoDb says pack of tens are also prefered
	packOfTens, ilay := [][]string{{}}, int(0)
	for _, ll := range layers {
		for _, l := range ll {
			if len(packOfTens[ilay]) < 10 {
				packOfTens[ilay] = append(packOfTens[ilay], l)
			} else {
				packOfTens = append(packOfTens, []string{})
				ilay++
				packOfTens[ilay] = append(packOfTens[ilay], l)
			}
		}
	}

	return packOfTens

}

func calcLayers(a area) [][]string {
	// center := a.Center(nil)
	clat, clon := a.Center.Lat, a.Center.Lon
	centerHash := geohash.EncodeWithPrecision(clat, clon, precision)
	layers := [][]string{{centerHash}}
	flat := []string{centerHash}

	var getLayers func([]string) [][]string
	getLayers = func(l []string) [][]string {
		curLayer := make([]string, 0)
		for _, e := range l {
			nbs := geohash.Neighbors(e)
			for _, nb := range nbs {
				if !utils.Contains(nb, flat) && validHash3(a, nb) {
					flat = append(flat, nb)
					curLayer = append(curLayer, nb)
				}
			}
		}

		nValid := len(curLayer)
		if nValid > 0 {
			layers = append(layers, curLayer)
		} else {
			return layers
		}

		if nValid > len(l) {
			return getLayers(curLayer)
		} else {
			return layers
		}

	}

	getLayers(layers[0])

	// required for firestore...
	// mongoDb says pack of tens are also prefered
	packOfTens, ilay := [][]string{{}}, int(0)
	for _, ll := range layers {
		for _, l := range ll {
			if len(packOfTens[ilay]) < 10 {
				packOfTens[ilay] = append(packOfTens[ilay], l)
			} else {
				packOfTens = append(packOfTens, []string{})
				ilay++
				packOfTens[ilay] = append(packOfTens[ilay], l)
			}
		}
	}

	return packOfTens

}

type area struct {
	Center latlon   `msgpack:"center"`
	Perim  []latlon `msgpack:"perim"`
}

type boostRequest struct {
	BoostId       string   `msgpack:"boostId"`
	Token         string   `msgpack:"token"`
	DeviceID      string   `msgpack:"deviceId"`
	SenderID      string   `msgpack:"senderId"`
	ChangeAddress []byte   `msgpack:"changeAddr"`
	S1            []byte   `msgpack:"s1"`
	PricePerHead  int      `msgpack:"pph"`
	InputSats     int      `msgpack:"inputSats"`
	PartialTx     []byte   `msgpack:"tx"`
	Limit         int      `msgpack:"limit"`
	MaxAge        int      `msgpack:"maxAge"`
	MinAge        int      `msgpack:"minAge"`
	Genders       []string `msgpack:"genders"` // "male", "female", ""
	Interests     []string `msgpack:"interests"`
	Areas         []area   `msgpack:"areas"`
	BoostMessage  []byte   `msgpack:"boostMessage"`
	FullMedia     []byte   `msgpack:"fullMedia"`
	// Media         map[string]string      `msgpack:"boostMedia"`
	// MediaPayload  []byte                 `msgpack:"mediaPayload"`
}

func (br *boostRequest) Query(layer []string, lim int) (cur *mongo.Cursor, err error) {
	filter := bson.M{
		"$and": bson.A{
			bson.M{"age": bson.M{"$lte": br.MaxAge}},
			bson.M{"age": bson.M{"$gte": br.MinAge}},
			bson.M{"gender": bson.M{"$in": br.Genders}},
			bson.M{"geohash": bson.M{"$in": layer}},
			bson.M{"interests": bson.M{"$elemMatch": bson.M{"$in": br.Interests}}},
		},
	}

	opts := options.Find().SetLimit(int64(lim))
	return db.Users.Find(context.Background(), filter, opts)
}

type user struct {
	Id           string  `bson:"_id"`
	MainDeviceId string  `bson:"mainDeviceId"`
	MediaId      string  `bson:"mediaId"`
	Place        string  `bson:"place"`
	Lat          float64 `bson:"latitude"`
	Lon          float64 `bson:"longitude"`
	Token        string  `bson:"token"`
	Neuter       string  `bson:"neuter"`
}

var chat_server_router = map[string]string{}
var media_server_router = map[string]string{}

func writeBoosts(users map[string][]*user, mediaPlaces []string, br *boostRequest) {
	writeTheMedia := func(mediaId, mediaPlace string, fullMedia []byte, ch chan error) {
		ip := media_server_router[mediaPlace]
		url := ip + "/media/" + mediaId + "/true"
		ct := "application/octet-stream"
		res, err := http.DefaultClient.Post(url, ct, bytes.NewReader(fullMedia))
		if err != nil {
			ch <- err
		} else if res.StatusCode != 200 {
			ch <- fmt.Errorf("Write media to place=%s failed with code=%d",
				mediaPlace, res.StatusCode)
		} else {
			ch <- nil
		}
	}

	writeTheBoosts := func(chatPlace string, usrs []*user, ch chan error) {
		ip := chat_server_router[chatPlace]
		conn, err := net.Dial("tcp", ip)
		if err != nil {
			ch <- err
			return
		}
		defer conn.Close()
		lenbuf := make([]byte, 2)
		binary.BigEndian.PutUint16(lenbuf, uint16(len(br.BoostMessage)))
		_, err0 := conn.Write([]byte{0x88})
		_, err1 := conn.Write(lenbuf)
		_, err2 := conn.Write(br.BoostMessage)
		if err = errors.Join(err0, err1, err2); err != nil {
			ch <- err
			return
		}

		makeBoostTag := func(bb *bytes.Buffer, u *user) []byte {
			bb.WriteString("b-")
			bb.WriteString(utils.MakeTimestampStr())
			bb.WriteByte('-')
			bb.WriteString(u.Id)
			return bb.Bytes()
		}

		bb := new(bytes.Buffer)
		for _, usr := range usrs {
			boostTag := makeBoostTag(bb, usr)
			binary.BigEndian.PutUint16(lenbuf, uint16(len(boostTag)))
			_, err0 = conn.Write(lenbuf)
			_, err1 = conn.Write(boostTag)
			if err = errors.Join(err0, err1); err != nil {
				ch <- err // TODO need actual error handling
				return
			}
		}
	}

	chatch := make(chan error, len(users))
	for cp, usrs := range users {
		go writeTheBoosts(cp, usrs, chatch)
	}

	mediach := make(chan error, len(mediaPlaces))
	if len(br.FullMedia) > 0 {
		mediach = make(chan error, len(mediaPlaces))
		mm := flatgen.GetRootAsMediaMetadata(br.FullMedia, 2)
		mid := utils.FastBytesToString(mm.TimeId())
		for _, mp := range mediaPlaces {
			go writeTheMedia(mid, mp, br.FullMedia, mediach)
		}
	}

	var chlen int
	if mediach == nil {
		chlen = len(users)
	} else {
		chlen = len(users) + len(mediaPlaces)
	}

	for i := 0; i < chlen; i++ {
		select {
		case err := <-chatch:
			if err != nil {
				// TODO
			}
		case err := <-mediach:
			if err != nil {
				// TODO
			}
		}
	}

}

func satsPrefix(sats int) string {
	// let's set an upper limit of pph of 1bsv // which is way over anyone will pay
	// that sets 100 000 000 sats
	// const upsat uint64 = 100000000

	// max is 4bill for a single head, which is like 42 bsv
	const upper uint32 = math.MaxUint32
	dif := upper - uint32(sats)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, dif)
	prfx := base32.HexEncoding.EncodeToString(buf.Bytes())

	return prfx
}

func HandleBoostRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var br boostRequest
	if err := msgpack.NewDecoder(r.Body).Decode(&br); err != nil {
		w.WriteHeader(500)
		log.Println("HandleBoostRequest error decoding boostRequest:", err)
		return
	}

	if br.PricePerHead > math.MaxUint32 {
		w.WriteHeader(500)
		log.Println("price per head exceeds maximun amount of 42 bsv:", br.PricePerHead)
		return
	}

	tx := TxFromRdr(bytes.NewReader(br.PartialTx))
	log.Println("tx pre boost:", tx.Formatted())

	lim := br.Limit
	// users := make([]*user, 0, lim)
	musers := make(map[string][]*user)
	mediaPlaces := make([]string, 0, 10)

	for _, a := range br.Areas {
		usrs, mp, newlim := scanArea(ctx, &br, a, lim)
		utils.AddAllToSet(mediaPlaces, mp...)
		for place, us := range usrs {
			musers[place] = append(musers[place], us...)
		}
		// users = append(users, usrs...)
		lim = newlim
		if lim == 0 {
			break
		}
	}

	nOuts := br.Limit - lim
	if nOuts == 0 {
		log.Println("Haven't found any people to boost")
		w.WriteHeader(500)
		return
	}

	rdyTx := BoostScript(tx, br.S1, nOuts, br.PricePerHead, br.InputSats, br.ChangeAddress)
	rawTx := rdyTx.Raw()
	rawTxHex := hex.EncodeToString(rawTx) // txid := Txid(rawTx)
	// txidHex := hex.EncodeToString(txid)
	log.Printf("ready tx\n%v", rdyTx.Formatted())
	// rawTxHex := hex.EncodeToString(rdyTx.Raw())
	log.Printf("raw hex tx\n%v\n", rawTxHex)
	// txHexRdr := strings.NewReader(rawTxHex)
	txPayload := map[string]any{"txhex": rawTxHex}
	// txPayload := map[string]interface{}{"rawTx": rawTxHex}
	// txPayload := map[string]interface{}{"raw": rawTxHex}
	rawPayload, err := json.Marshal(txPayload)
	if err != nil {
		log.Println("err marshalling txPayload:", err)
		w.WriteHeader(500)
		return
	}
	payloadRdr := bytes.NewReader(rawPayload)

	// const url string = "https://test-api.bitails.io/tx/broadcast"
	const url string = "https://api.whatsonchain.com/v1/bsv/test/tx/raw"
	// const url string = "https://api.taal.com/api/v1/broadcast"
	// req, err := http.NewRequest("POST", url, payloadRdr)
	// req.Header = map[string][]string{
	// 	"Content-Type": {"application/json"},
	// 	"Authorization": {"Bearer " + taal_api_key},
	// }
	// rsp, err := http.DefaultClient.Do(req)

	rsp, err := http.Post(url, "application/json", payloadRdr)
	if err != nil {
		w.WriteHeader(500)
		log.Println("error making broadcast request:", err)
		return
	}

	if rsp.StatusCode != 200 {
		rbuf, _ := io.ReadAll(rsp.Body)
		log.Println("error broadcasting tx:", string(rbuf))
		w.WriteHeader(500)
		return
	}

	var rjson map[string]any
	err = json.NewDecoder(rsp.Body).Decode(&rjson)
	if err != nil {
		w.WriteHeader(500)
		log.Println("error decoding response:", err)
		return
	}

	status, ok := rjson["status"].(int)
	if !ok {
		log.Println("error finding response status")
		w.WriteHeader(500)
		return
	}

	if status != 200 {
		title, detail := rjson["title"], rjson["detail"]
		log.Println("%s\n%d\n%s\n", title, status, detail)
		w.WriteHeader(500)
		return
	}

	writeBoosts(musers, mediaPlaces, &br)

	header, body := "Completed Boost", fmt.Sprintf("Found %v targets", nOuts)
	if _, err = sendNotification(header, body, br.Token); err != nil {
		log.Println("error pushing notification to booster:", err)
	}
}

func scanArea(ctx context.Context,
	b *boostRequest, a area, lim int) (map[string][]*user, []string, int) {
	layers := calcLayers2(a)
	fmt.Printf("layers: %v\n", layers)

	// users := make([]*user, 0, lim)
	musers := make(map[string][]*user)
	mediaPlaces := make([]string, 0, 3)

	var curlim int = lim

	for _, layer := range layers {
		la, li := layer, curlim
		cursor, err := b.Query(la, li)
		if err != nil {
			log.Println("error making a query:", err)
			continue
		}
		for cursor.Next(ctx) {
			var usr user
			if err := bson.Unmarshal(cursor.Current, &usr); err != nil {
				break
			}

			cl2 := closest2(a.Perim, latlon{Lat: usr.Lat, Lon: usr.Lon})
			usrDist := geoDist(latlon{Lat: usr.Lat, Lon: usr.Lon}, a.Center)

			// check if the user is indeed within the area
			// because someone can be in a geohash, but not the actual area
			valid := utils.Any(cl2, func(ll latlon) bool { return usrDist <= ll.RefDist })

			if valid {
				musers[usr.Place] = append(musers[usr.Place], &usr)
				mp := utils.ExtractMediaPlace(usr.MediaId)
				mediaPlaces = append(mediaPlaces, mp)
				curlim--
				if curlim == 0 {
					break
				}
			}
		}

		if curlim == 0 {
			break
		}
	}
	fmt.Printf("we found %v users for the boost\n", lim-curlim)
	return musers, mediaPlaces, curlim
}
