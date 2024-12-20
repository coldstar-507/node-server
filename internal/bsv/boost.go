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
	"log"
	"math"
	"net"
	"net/http"
	"sync"

	"firebase.google.com/go/v4/messaging"
	"github.com/coldstar-507/flatgen"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/router/router_utils"
	"github.com/coldstar-507/utils/id_utils"
	"github.com/coldstar-507/utils/utils"
	"github.com/mmcloughlin/geohash"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const taal_api_key = "testnet_3860616b1cf1bb23110db44440f65899"

type LatLon struct {
	Lat     float64 `msgpack:"lat" json:"lat"`
	Lon     float64 `msgpack:"lon" json:"lon"`
	RefDist float64 `msgpack:"refDist" json:"refDist"`
}

const GeohashPrecision = 4

func SendNotification(header, body, token string) (string, error) {
	return db.Messager.Send(context.Background(), &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: header,
			Body:  body,
		}})
}

func GeoDist(ll1, ll2 LatLon) float64 {
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

func geoBearing(ll1, ll2 LatLon) float64 {
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

func closest2(l []LatLon, p LatLon) []LatLon {
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
	return []LatLon{l[ix], l[ix2]}
}

func validHash(a *Area, hash string) bool {
	box := geohash.BoundingBox(hash)
	bclat, bclon := box.Center()
	boxCenter := LatLon{Lat: bclat, Lon: bclon}
	bounds := []LatLon{
		{Lat: box.MaxLat, Lon: box.MaxLng},
		{Lat: box.MaxLat, Lon: box.MinLng},
		{Lat: box.MinLat, Lon: box.MaxLng},
		{Lat: box.MinLat, Lon: box.MinLng},
	}

	twoClosest := closest2(a.Perim, boxCenter)
	for _, b := range bounds {
		valid := utils.Any(twoClosest, func(ll LatLon) bool {
			return GeoDist(b, a.Center) < ll.RefDist
		})
		if valid {
			return true
		}
	}
	return false
}

func MakeGeohash(ll LatLon) string {
	return geohash.EncodeWithPrecision(ll.Lat, ll.Lon, GeohashPrecision)
}

func CalcLayers(a *Area) [][]string {
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
				if !utils.Contains(nb, flat) && validHash(a, nb) {
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
// 				if !utils.Contains(nb, flat) && validHash3(a, nb) {
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

type Area struct {
	Center LatLon   `msgpack:"center" json:"center"`
	Perim  []LatLon `msgpack:"perim" json:"perim"`
}

type BoostTest struct {
	BoostId       string   `json:"boostId"`
	Token         string   `json:"token"`
	DeviceId      uint32   `json:"deviceId"`
	SenderId      string   `json:"senderId"`
	ChangeAddress string   `json:"changeAddr"`
	S1            string   `json:"s1"`
	PricePerHead  int      `json:"pph"`
	InputSats     int      `json:"inputSats"`
	PartialTx     string   `json:"tx"`
	Limit         int      `json:"limit"`
	MaxAge        int      `json:"maxAge"`
	MinAge        int      `json:"minAge"`
	Genders       []string `json:"genders"`
	Interests     []string `json:"interests"`
	Areas         []*Area  `json:"areas"`
	BoostMessage  string   `json:"boostMessage"`
	// FullMedia     []byte   `msgpack:"fullMedia"`
}

type BoostRequest struct {
	BoostId       string   `msgpack:"boostId"`
	Token         string   `msgpack:"token"`
	DeviceId      string   `msgpack:"deviceId"`
	SenderId      string   `msgpack:"senderId"`
	ChangeAddress []byte   `msgpack:"changeAddr"`
	S1            []byte   `msgpack:"s1"`
	PricePerHead  int      `msgpack:"pph"`
	InputSats     int      `msgpack:"inputSats"`
	PartialTx     []byte   `msgpack:"tx"`
	Limit         int      `msgpack:"limit"`
	MaxAge        int      `msgpack:"maxAge"`
	MinAge        int      `msgpack:"minAge"`
	Genders       []string `msgpack:"genders"`
	Interests     []string `msgpack:"interests"`
	Areas         []*Area  `msgpack:"areas"`
	BoostMessage  []byte   `msgpack:"boostMessage"`
	FullMedia     []byte   `msgpack:"fullMedia"`
	// Media         map[string]string      `msgpack:"boostMedia"`
	// MediaPayload  []byte                 `msgpack:"mediaPayload"`
}

func BoostQuery(layer, genders, interests []string,
	lim, maxAge, minAge int) (cur *mongo.Cursor, err error) {
	log.Printf(`
ScanAreas:
  layer:     %v
  genders:   %v
  interests: %v
  limit:     %v
  minAge:    %v
  maxAge:    %v
`, layer, genders, interests, lim, minAge, maxAge)

	filter_ := bson.M{
		"type":    "user",
		"age":     bson.M{"$lte": maxAge, "$gte": minAge},
		"geohash": bson.M{"$in": layer},
	}
	if len(genders) > 0 {
		filter_["gender"] = bson.M{"$in": genders}
	}
	if len(interests) > 0 {
		filter_["interests"] = bson.M{"$elemMatch": bson.M{"$in": interests}}
	}
	b, _ := json.MarshalIndent(filter_, "", "    ")
	log.Println("query:\n", string(b))

	// opts := options.Find().SetLimit(int64(lim))
	return db.Nodes.Find(context.Background(), filter_)
	// return db.Nodes.Find(context.Background(), filter_, opts)
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

func WriteBoosts(users map[uint16][]*User,
	interests []string, boostMessage, fullMedia []byte, pph int) {

	ts := utils.MakeTimestamp()
	relMediaplaces := make(map[uint16]uint16)
	for p := range users {
		rm := router_utils.ChatRouter().RelativeMedias(router_utils.Uint16ToHex(p))
		relMediaplaces[p] = router_utils.HexToUint16(rm[0])
	}
	mediaPlaces := make([]uint16, 0, len(relMediaplaces))
	for _, mp := range relMediaplaces {
		mediaPlaces, _ = utils.AddToSet(mp, mediaPlaces)
	}

	// each write media compete with each other on memory, so can't be concurrent here
	writeTheMedia := func(mediaPlace uint16, fullMedia []byte) error {
		fm := flatgen.GetRootAsFullMedia(fullMedia, 0)
		mt := fm.Metadata(nil)
		ref := mt.Ref(nil)
		utils.Assert(ref != nil, "boost media ref can't be null")
		ref.MutatePlace(mediaPlace)
		ref.MutatePermanent(false)
		ref.MutateTimestamp(utils.MakeTimestamp())
		refHex := hex.EncodeToString(id_utils.MakeRawMediaRef(ref))
		hexMediaPlace := router_utils.Uint16ToHex(mediaPlace)
		ip := router_utils.MediaRouter().HostAndPort(hexMediaPlace)
		url := "http://" + ip + "/media/" + refHex
		ct := "application/octet-stream"
		res, err := http.DefaultClient.Post(url, ct, bytes.NewReader(fullMedia))

		if err != nil {
			log.Println("WriteBoosts: writeTheMedia:", err)
			return err
		} else if res.StatusCode != 200 {
			err = fmt.Errorf("Write media to place=%s failed with code=%d",
				hexMediaPlace, res.StatusCode)
			log.Println("WriteBoosts: writeTheMedia:", err)
			return err
		} else {
			return nil
		}
	}

	// similarly, writeTheBoosts also compete on memory
	writeTheBoosts := func(chatPlace uint16, usrs []*User) error {
		hexPlace := router_utils.Uint16ToHex(chatPlace)
		ip := router_utils.ChatRouter().Host(hexPlace) + ":11003"
		log.Printf("WriteBoost: writeTheBoost: dialing tcp@%s\n", ip)
		conn, err := net.Dial("tcp", ip)
		if err != nil {
			log.Printf("WriteBoost: writeTheBoost: error dial tcp@%s: %v", ip, err)
			return err
		}
		defer conn.Close()

		bm := flatgen.GetRootAsMessageEvent(boostMessage, 0)
		if mr := bm.MediaRef(nil); mr != nil {
			mr.MutatePlace(relMediaplaces[chatPlace])
			mr.MutateTimestamp(ts)
		}

		calcBoostTagLen := func() uint16 {
			var l uint16 = 1 + 8 + 4 + 13
			for _, x := range interests {
				l += uint16(1 + len(x))
			}
			return l
		}
		boostTagLen := calcBoostTagLen()

		err0 := binary.Write(conn, binary.BigEndian, byte(0x88))
		err1 := binary.Write(conn, binary.BigEndian, uint16(len(boostMessage)))
		_, err2 := conn.Write(boostMessage)
		err3 := binary.Write(conn, binary.BigEndian, boostTagLen)
		err4 := binary.Write(conn, binary.BigEndian, uint32(len(usrs)))
		if err = errors.Join(err0, err1, err2, err3, err4); err != nil {
			log.Println("WriteBoost: writeTheBoost: error writing boostMsg:", err)
			return err
		}

		log.Printf(`WriteBoost: writeTheBoost:
t           : %x
msgLen      : %d
boostTagLen : %d
nBoosts     : %d
`, 0x88, len(boostMessage), boostTagLen, len(usrs))

		// boost tags from a specific boost have the same length
		// {boost byte}|{boost ts}|{raw user nodeId}|( ... {len interest}|{interest} )

		// makeBoostTag := func(bb *bytes.Buffer, u *User) []byte {
		// 	bb.Reset()
		// 	bb.WriteByte(id_utils.KIND_BOOST)                         // 1
		// 	binary.Write(bb, binary.BigEndian, utils.MakeTimestamp()) // 8
		// 	binary.Write(bb, binary.BigEndian, uint32(pph))           // 4
		// 	log.Printf("WriteBoost: writeTheBoost: avail len=%d\n", bb.Available())
		// 	log.Printf("WriteBoost: writeTheBoost: cap=%d\n", bb.Cap())
		// 	hex.Decode(bb.AvailableBuffer(), []byte(u.Id)) // 13
		// 	for _, x := range interests {
		// 		bb.WriteByte(byte(len(interests)))
		// 		bb.WriteString(x)
		// 	}
		// 	return bb.Bytes()
		// }

		// this slightly more convoluated way is prefered because
		// there is no good way to decode hex into a writer without allocation
		writeBoostTag := func(buf []byte, u *User) {
			buf[0] = id_utils.KIND_BOOST                                       // 1
			binary.BigEndian.PutUint64(buf[1:], uint64(utils.MakeTimestamp())) // 8
			binary.BigEndian.PutUint32(buf[9:], uint32(pph))                   // 4
			i, err := hex.Decode(buf[13:], []byte(u.Id))                       // 13
			if err != nil {
				panic(err)
			} else if i != 13 {
				panic("i != 13")
			}
			var n = 26
			for _, x := range interests {
				l := len(x)
				buf[n] = byte(l)
				copy(buf[n+1:], []byte(x))
				n += 1 + l
			}
		}

		log.Printf("WriteBoost: writeTheBoost: boostTagLen=%d\n", boostTagLen)

		buf := make([]byte, boostTagLen)
		// bb := bytes.NewBuffer(make([]byte, 0, boostTagLen))
		for _, usr := range usrs {
			writeBoostTag(buf, usr)
			if _, err = conn.Write(buf); err != nil {
				log.Println("WriteBoosts: writeTheBoost: err writing tag:", err)
				return err
			}
			// if _, err = conn.Write(makeBoostTag(bb, usr)); err != nil {
			// 	log.Println("WriteBoosts: writeTheBoost: err writing tag:", err)
			// 	return err
			// }
		}

		var r byte
		if err := binary.Read(conn, binary.BigEndian, &r); err != nil {
			log.Println("WriteBoosts: writeTheBoost: error reading completion:\n\t",
				err)

		} else if r == 0x88 {
			log.Println("WriteBoosts: writeTheBoost: DONE")
		} else {
			log.Printf("WriteBoosts: writeTheBoost: wrong completion byte: %x\n", r)
		}
		return nil
	}

	wg := new(sync.WaitGroup)

	if len(fullMedia) > 0 {
		wg.Add(1)
		go func() {
			for _, mp := range mediaPlaces {
				writeTheMedia(mp, fullMedia)
			}
			wg.Done()
		}()
	}

	wg.Add(1)
	go func() {
		for cp, usrs := range users {
			writeTheBoosts(cp, usrs)
		}
		wg.Done()
	}()
	wg.Wait()
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

func ScanArea(ctx context.Context, a *Area,
	genders, interests []string, minAge, maxAge, lim int) (map[uint16][]*User, int) {

	layers := CalcLayers(a)

	musers := make(map[uint16][]*User)
	var curlim = lim

	for _, layer := range layers {
		la, li := layer, curlim
		cursor, err := BoostQuery(la, genders, interests, li, maxAge, minAge)
		if err != nil {
			log.Println("error making a query:", err)
			continue
		}

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

			cl2 := closest2(a.Perim, LatLon{Lat: usr.Lat, Lon: usr.Lon})
			usrDist := GeoDist(LatLon{Lat: usr.Lat, Lon: usr.Lon}, a.Center)

			// check if the user is indeed within the area
			// because someone can be in a geohash, but not the actual area
			valid := utils.Any(cl2, func(ll LatLon) bool {
				return usrDist <= ll.RefDist
			})

			if valid {
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
