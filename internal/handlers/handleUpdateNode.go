package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/coldstar-507/node-server/internal/db"
	// "github.com/coldstar-507/utils/utils"
	// "github.com/jackc/pgxutil"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// func UpdateNodeById(id string, update map[string]any) error {
// 	return pgxutil.UpdateRow(context.Background(),
// 		db.Pool, "nodes", update, map[string]any{"id": id})
// }

func UpdateByIdsMongo(coll *mongo.Collection, ids []string, update bson.Raw) error {
	filter := bson.M{"_id": bson.M{"$in": ids}}
	_, err := coll.UpdateMany(context.Background(), filter, update)
	return err
}

func UpdateByIdMongo(coll *mongo.Collection, id string, update interface{}) error {
	_, err := coll.UpdateByID(context.Background(), id, update)
	return err
}

var ErrNoMatchTs = errors.New("No match, likely didn't match timestamp")

func UpdateMatchTS(coll *mongo.Collection, id string, ts time.Time, update any) error {
	ts = ts.Truncate(time.Millisecond)
	filter := bson.M{"_id": id, "$or": bson.A{
		bson.M{"lastUpdate": ts},
		bson.M{"lastUpdate": nil},
	}}
	r, err := coll.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return err
	}
	if r.MatchedCount == 0 {
		return ErrNoMatchTs
	}
	return nil
}

// var NodeUpdatableFields = []string{
// 	"name",
// 	"mediaRef",
// 	"backgroundRef",
// 	"interests",
// 	"latitude",
// 	"longitude",
// 	"verified",
// 	"birthday",
// 	"gender",
// 	"interests",
// 	"geohash",
// 	"messagingTokens",
// 	"description",
// 	"website",
// 	"children",
// 	"countryCode",
// 	"themes",
// 	"videos",
// 	"images",
// 	"anims",
// 	"phone",
// 	"blocks",
// 	"conns",
// }

func HandleUpdateNodeById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tsStr := r.PathValue("ts")
	tsInt, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		w.WriteHeader(504)
		return
	}

	ts := time.UnixMilli(tsInt)
	tsFmt := ts.Format("2006-01-02T15:04:05.000Z")
	log.Printf("HandleNodeUpdate(%s, %d) -> %s\n", id, tsInt, tsFmt)
	var updateReq map[string]any
	if err := msgpack.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		log.Println("HandleUpdateNode error decoding msgpack body:", err)
		w.WriteHeader(500)
	} else {
		// for k := range updateReq {
		// 	if !utils.Contains(k, NodeUpdatableFields) {
		// 		log.Println("HandleUpdateNode: WARNING: invalid update key:", k)
		// 		delete(updateReq, k)
		// 	}
		// }

		// if len(updateReq) > 0 {
		update := bson.M{
			"$set":         updateReq,
			"$currentDate": bson.M{"lastUpdate": true},
		}

		if err = UpdateMatchTS(db.Nodes, id, ts, update); err != nil {
			log.Println("HandleUpdateNode error updating node:", err)
			if errors.Is(err, ErrNoMatchTs) {
				w.WriteHeader(501)
			} else {
				w.WriteHeader(502)
			}
		}
		// }
	}
}
