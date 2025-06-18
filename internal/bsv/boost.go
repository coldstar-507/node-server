package bsv

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"time"

	"firebase.google.com/go/v4/messaging"
	fg "github.com/coldstar-507/flatgen"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/router-server/router_utils"
	"github.com/coldstar-507/utils2"
	// "github.com/coldstar-507/utils/id_utils"
	// "github.com/coldstar-507/utils/utils"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/mmcloughlin/geohash"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// const taal_api_key = "testnet_3860616b1cf1bb23110db44440f65899"

type chat_place = uint16
type node_place = uint16
type media_place = uint16

const GeohashPrecision = 4

func SendNotification(header, body, token string) (string, error) {
	return db.Messager.Send(context.Background(), &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: header,
			Body:  body,
		}})
}

func GeoDist(ll1, ll2 *fg.LatLonT) float64 {
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

func geoBearing(ll1, ll2 *fg.LatLon) float64 {
	lat1, lon1, lat2, lon2 := ll1.Lat(), ll1.Lon(), ll2.Lat(), ll2.Lon()
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

// func closest(l []latlon, p latlon) latlon {
// 	ix, smallDist := int(0), math.MaxFloat64
// 	for i, x := range l {
// 		dist := geoDist(x, p)
// 		if dist < smallDist {
// 			smallDist = dist
// 			ix = i
// 		}
// 	}
// 	return l[ix]
// }

func closest2(l []*fg.LatLonT, p *fg.LatLonT) []*fg.LatLonT {
	ix, smallDist := int(0), math.MaxFloat64
	ix2, smallDist2 := int(0), math.MaxFloat64
	for i, x := range l {
		dist := GeoDist(x, p)
		if dist < smallDist {
			smallDist = dist
			ix = i
		}
		if dist > smallDist && dist < smallDist2 {
			smallDist2 = dist
			ix2 = i
		}
	}
	return []*fg.LatLonT{l[ix], l[ix2]}
}

func validHash(a *fg.AreaT, hash string) bool {
	box := geohash.BoundingBox(hash)
	bclat, bclon := box.Center()
	boxCenter := &fg.LatLonT{Lat: bclat, Lon: bclon}
	bounds := []*fg.LatLonT{
		{Lat: box.MaxLat, Lon: box.MaxLng},
		{Lat: box.MaxLat, Lon: box.MinLng},
		{Lat: box.MinLat, Lon: box.MaxLng},
		{Lat: box.MinLat, Lon: box.MinLng},
	}

	twoClosest := closest2(a.Perim, boxCenter)
	for _, b := range bounds {
		valid := utils2.Any(twoClosest, func(ll *fg.LatLonT) bool {
			return GeoDist(b, a.Center) < ll.RefDist
		})
		if valid {
			return true
		}
	}
	return false
}

func MakeGeohash(ll *fg.LatLonT) string {
	return geohash.EncodeWithPrecision(ll.Lat, ll.Lon, GeohashPrecision)
}

func CalcLayers(a *fg.AreaT) [][]string {
	clat, clon := a.Center.Lat, a.Center.Lon
	centerHash := geohash.EncodeWithPrecision(clat, clon, GeohashPrecision)
	layers := [][]string{{centerHash}}
	flat := []string{centerHash}

	var getLayers func([]string) [][]string
	getLayers = func(l []string) [][]string {
		curLayer := make([]string, 0)
		for _, e := range l {
			nbs := geohash.Neighbors(e)
			for _, nb := range nbs {
				if !utils2.Contains(nb, flat) && validHash(a, nb) {
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

// func calcLayers(a area) [][]string {
// 	// center := a.Center(nil)
// 	clat, clon := a.Center.Lat, a.Center.Lon
// 	centerHash := geohash.EncodeWithPrecision(clat, clon, precision)
// 	layers := [][]string{{centerHash}}
// 	flat := []string{centerHash}

// 	var getLayers func([]string) [][]string
// 	getLayers = func(l []string) [][]string {
// 		curLayer := make([]string, 0)
// 		for _, e := range l {
// 			nbs := geohash.Neighbors(e)
// 			for _, nb := range nbs {
// 				if !utils2.Contains(nb, flat) && validHash3(a, nb) {
// 					flat = append(flat, nb)
// 					curLayer = append(curLayer, nb)
// 				}
// 			}
// 		}

// 		nValid := len(curLayer)
// 		if nValid > 0 {
// 			layers = append(layers, curLayer)
// 		} else {
// 			return layers
// 		}

// 		if nValid > len(l) {
// 			return getLayers(curLayer)
// 		} else {
// 			return layers
// 		}

// 	}

// 	getLayers(layers[0])

// 	// required for firestore...
// 	// mongoDb says pack of tens are also prefered
// 	packOfTens, ilay := [][]string{{}}, int(0)
// 	for _, ll := range layers {
// 		for _, l := range ll {
// 			if len(packOfTens[ilay]) < 10 {
// 				packOfTens[ilay] = append(packOfTens[ilay], l)
// 			} else {
// 				packOfTens = append(packOfTens, []string{})
// 				ilay++
// 				packOfTens[ilay] = append(packOfTens[ilay], l)
// 			}
// 		}
// 	}

// 	return packOfTens
// }

// type Area struct {
// 	Center LatLon   `msgpack:"center" json:"center"`
// 	Perim  []LatLon `msgpack:"perim" json:"perim"`
// }

// type BoostTest struct {
// 	BoostId       string   `json:"boostId"`
// 	Token         string   `json:"token"`
// 	DeviceId      uint32   `json:"deviceId"`
// 	SenderId      string   `json:"senderId"`
// 	ChangeAddress string   `json:"changeAddr"`
// 	S1            string   `json:"s1"`
// 	PricePerHead  int      `json:"pph"`
// 	InputSats     int      `json:"inputSats"`
// 	PartialTx     string   `json:"tx"`
// 	Limit         int      `json:"limit"`
// 	MaxAge        int      `json:"maxAge"`
// 	MinAge        int      `json:"minAge"`
// 	Genders       []string `json:"genders"`
// 	Interests     []string `json:"interests"`
// 	Areas         []*Area  `json:"areas"`
// 	BoostMessage  string   `json:"boostMessage"`
// 	// FullMedia     []byte   `msgpack:"fullMedia"`
// }

// type BoostRequest struct {
// 	BoostId       string   `msgpack:"boostId"`
// 	Token         string   `msgpack:"token"`
// 	DeviceId      string   `msgpack:"deviceId"`
// 	SenderId      string   `msgpack:"senderId"`
// 	ChangeAddress []byte   `msgpack:"changeAddr"`
// 	S1            []byte   `msgpack:"s1"`
// 	PricePerHead  int      `msgpack:"pph"`
// 	InputSats     int      `msgpack:"inputSats"`
// 	PartialTx     []byte   `msgpack:"tx"`
// 	Limit         int      `msgpack:"limit"`
// 	MaxAge        int      `msgpack:"maxAge"`
// 	MinAge        int      `msgpack:"minAge"`
// 	Genders       []string `msgpack:"genders"`
// 	Interests     []string `msgpack:"interests"`
// 	Areas         []*Area  `msgpack:"areas"`
// 	BoostMessage  []byte   `msgpack:"boostMessage"`
// 	FullMedia     []byte   `msgpack:"fullMedia"`
// 	// Media         map[string]string      `msgpack:"boostMedia"`
// 	// MediaPayload  []byte                 `msgpack:"mediaPayload"`
// }

func BoostQuery(layer []string, lim uint64, q *fg.BoostQueryT) (cur *mongo.Cursor, err error) {
	log.Printf(`
ScanAreas:
  layer:     %v
  genders:   %v
  interests: %v
  countries: %v
  limit:     %v
  minAge:    %v
  maxAge:    %v
`, layer, q.Genders, q.Interests, q.Countries, lim, q.MinAge, q.MaxAge)

	now := time.Now()
	youngestAge := time.Hour * time.Duration(24.0*365.25) * time.Duration(q.MinAge)
	oldestAge := time.Hour * time.Duration(24*365.25) * time.Duration(q.MaxAge)
	youngestBday := now.Add(-youngestAge).UnixMilli()
	oldestBday := now.Add(-oldestAge).UnixMilli()

	m := bson.M{}
	if len(q.Countries) > 0 {
		m["countryCode"] = bson.M{"$in": q.Countries}
	}
	if len(layer) > 0 {
		m["geohash"] = bson.M{"$in": layer}
	}
	if len(q.Genders) > 0 && len(q.Genders) != 3 {
		m["gender"] = bson.M{"$in": q.Genders}
	}
	m["birthday"] = bson.M{
		"$gte": oldestBday,
		"$lte": youngestBday,
	}
	if len(q.Interests) > 0 {
		m["interests"] = bson.M{
			"$elemMatch": bson.M{
				"$in": q.Interests,
			},
		}
	}

	b, _ := json.MarshalIndent(m, "", "    ")
	fmt.Println(string(b))

	opts := options.Find().SetLimit(int64(lim))
	return db.Nodes.Find(context.Background(), m, opts)
}

type User struct {
	Id           string   `bson:"_id"`
	MainDeviceId uint32   `bson:"mainDeviceId"`
	Interests    []string `bson:"interests"`
	ChatPlaces   []uint16 `bson:"chatPlaces"`
	Geohash      string   `bson:"geohash"`
	Age          int      `bson:"age"`
	Lat          float64  `bson:"latitude"`
	Lon          float64  `bson:"longitude"`
	Neuter       string   `bson:"neuter"`
	// Place        string   `bson:"place"`
	// Token        string   `bson:"token"`
}

func makeBooster(bld *flatbuffers.Builder, boost *fg.BoosterT, msgId *fg.MessageIdT,
	sats, utxoIx uint32, s1, rawNodeId, txid []byte, interests []string) []byte {
	bld.Reset()
	boost.Prefix = utils2.KIND_BOOST
	boost.Timestamp = 1 // cannot be 0 value here, it will be mutated
	boost.RawNodeId = rawNodeId
	boost.MsgId = msgId
	boost.Sats = sats
	boost.Secret = s1
	boost.Txid = txid
	boost.UtxoIx = utxoIx
	boost.Interests = interests
	bld.Finish(boost.Pack(bld))
	return bld.FinishedBytes()

}

func writeMedia(mp media_place, metadata []byte, temp *os.File) (*fg.MediaRefT, error) {
	defer temp.Seek(0, io.SeekStart) // reset file reader for other media uploads
	// strMediaPlace := router_utils.Uint16ToI(mp)
	ip := router_utils.MediaRouter().HostAndPort(mp)
	url := "http://" + ip + "/media/" //  + refHex
	ct := "application/octet-stream"

	lbuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lbuf, uint16(len(metadata)))
	body := io.MultiReader(bytes.NewReader(lbuf), bytes.NewReader(metadata), temp)

	res, err := http.DefaultClient.Post(url, ct, body)
	if err != nil {
		log.Println("WriteBoosts: writeTheMedia:", err)
		return nil, err
	} else if res.StatusCode != 200 {
		err = fmt.Errorf("Write media to place=%d failed with code=%d",
			mp, res.StatusCode)
		log.Println("WriteBoosts: writeTheMedia:", err)
		return nil, err
	} else { // will return a simple MediaRef
		ref := utils2.ReadRawMediaRef(res.Body)
		return ref, nil
	}
}

func WriteBoosts(users map[chat_place][]*User,
	br *fg.BoostRequest, interests []string, txid []byte, temp *os.File) {
	ln, f := utils2.Pln("WriteBoosts:"), utils2.Pf("WriteBoosts: ")

	boostMsgBytes, metadataBytes := br.BoostMessageBytes(), br.MediaBytes()
	relMediaplaces := make(map[chat_place]media_place)
	for p := range users {
		f("chat_place for relative medias=%d\n", p)
		rm := router_utils.ChatRouter().RelativeMedias(p)
		relMediaplaces[p] = rm[0]
	}

	var rawNodeId = utils2.NodeId{}

	relMediaRefs := make(map[chat_place]*fg.MediaRefT)

	boostMsg := fg.GetRootAsMessageEvent(boostMsgBytes, 0)
	bmMsgId := boostMsg.ChatId(nil)
	bmMsgIdRoot := bmMsgId.Root(nil)
	boostMsgMediaRef := boostMsg.MediaRef(nil)
	// similarly, writeTheBoosts also compete on memory
	writeTheBoosts := func(chatPlace chat_place, users_ []*User, utxoIx int) (int, error) {
		bmMsgIdRoot.MutateChatPlace(chatPlace)
		if temp != nil {
			if mp, exists := relMediaplaces[chatPlace]; exists {
				ref, hasRef := relMediaRefs[chatPlace]
				if !hasRef {
					ref_, err := writeMedia(mp, metadataBytes, temp)
					if err != nil {
						ln("writeMedia err:", err)
					}
					ref = ref_
				}
				boostMsgMediaRef.MutatePlace(ref.Place)
				boostMsgMediaRef.MutateTimestamp(ref.Timestamp)
			}
		}

		// strPlace := router_utils.Uint16ToI(chatPlace)
		ip := router_utils.ChatRouter().Host(chatPlace) + ":11003"
		f("dialing tcp@%s\n", ip)
		conn, err := net.Dial("tcp", ip)
		if err != nil {
			f("error dial tcp@%d: %v\n", ip, err)
			return utxoIx, err
		}
		defer conn.Close()

		err = utils2.WriteBin(conn, byte(0x88), uint16(len(boostMsgBytes)), boostMsgBytes,
			uint32(len(users_)))

		if err != nil {
			ln("error writing boostMsg:", err)
			return utxoIx, err
		}
		f("t=%x, msgLen=%d, nBoost=%d\n", 0x88, len(boostMsgBytes), len(users_))

		builder := flatbuffers.NewBuilder(1024)
		boosterT := &fg.BoosterT{}
		bmMsgIdT := bmMsgId.UnPack()
		for i := range len(users_) {
			hex.Decode(rawNodeId[:], []byte(users_[i].Id))
			booster := makeBooster(builder, boosterT, bmMsgIdT, br.PricePerHead(),
				uint32(i+utxoIx), br.S1Bytes(), rawNodeId[:], txid, interests)
			err := utils2.WriteBin(conn, uint16(len(booster)), booster)
			if err != nil {
				ln("err writing boost:", err)
				return i + utxoIx, err
			}
			utxoIx += 1
		}

		var r byte
		if err := utils2.ReadBin(conn, &r); err != nil {
			ln("err reading completion:", err)
		} else if r == 0x88 {
			ln("DONE")
		} else if r == 0x89 {
			var n uint32
			if err = utils2.ReadBin(conn, &n); err != nil {
				ln("err reading n booster wrote:", err)
			} else {
				f("completion err: %d booster wrote\n", n)
			}
		}
		return utxoIx, nil
	}

	var utxoIx = 1 // 0 is reserved for obligatory change utxo
	var err error
	for cp, usrs := range users {
		utxoIx, err = writeTheBoosts(cp, usrs, utxoIx)
		if err != nil {
			ln("TODO: writeTheBoosts err:", err)
		}
	}
}

func ScanArea(a *fg.AreaT, q *fg.BoostQueryT, lim uint64) (map[uint16][]*User, uint64) {
	layers := CalcLayers(a)

	musers := make(map[uint16][]*User)
	var curlim = lim

	for _, layer := range layers {
		la, li := layer, curlim
		cursor, err := BoostQuery(la, li, q)
		if err != nil {
			log.Println("error making a query:", err)
			continue
		}

		ctx := context.Background()
		for cursor.Next(ctx) {
			var usr User
			if err := bson.Unmarshal(cursor.Current, &usr); err != nil {
				log.Println("ScanArea: error unmarshalling bson:", err)
				break
			}
			if u, err := json.MarshalIndent(&usr, "", "    "); err != nil {
				log.Println("ScanArea: error pretty user:", err)
			} else {
				log.Println(string(u))
			}

			cl2 := closest2(a.Perim, &fg.LatLonT{Lat: usr.Lat, Lon: usr.Lon})
			usrDist := GeoDist(&fg.LatLonT{Lat: usr.Lat, Lon: usr.Lon}, a.Center)

			// check if the user is indeed within the area
			// because someone can be in a geohash, but not the actual area
			valid := utils2.Any(cl2, func(ll *fg.LatLonT) bool {
				return usrDist <= ll.RefDist
			})

			if valid {
				if len(usr.ChatPlaces) < 1 {
					log.Println("WARNING: ScanArea: user has no chat places")
					continue
				}
				usrCp := usr.ChatPlaces[0]
				musers[usrCp] = append(musers[usrCp], &usr)
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
	return musers, curlim
}
