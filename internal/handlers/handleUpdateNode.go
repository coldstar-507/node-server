package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/jackc/pgxutil"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func UpdateNodeById(id string, update map[string]any) error {
	return pgxutil.UpdateRow(context.Background(),
		db.Pool, "nodes", update, map[string]any{"id": id})
}

func UpdateByIdsMongo(coll *mongo.Collection, ids []string, update bson.Raw) error {
	filter := bson.M{"_id": bson.M{"$in": ids}}
	_, err := coll.UpdateMany(context.Background(), filter, update)
	return err
}

func UpdateByIdMongo(coll *mongo.Collection, id string, update interface{}) error {
	_, err := coll.UpdateByID(context.Background(), id, update)
	return err
}

func HandleUpdateNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var reqMap map[string]any
	updateMap := make(map[string]any)
	if err := msgpack.NewDecoder(r.Body).Decode(&reqMap); err != nil {
		log.Println("HandleUpdateNode error decoding msgpack body:", err)
		w.WriteHeader(500)
	} else {
		for _, x := range NodeUpdatableFields {
			if reqMap[x] != nil {
				updateMap[x] = reqMap[x]
			}
		}
		if len(updateMap) > 0 {
			update := map[string]any{"$set": updateMap}
			if err = UpdateByIdMongo(db.Nodes, id, update); err != nil {
				log.Println("HandleUpdateNode error updating node:", err)
				w.WriteHeader(500)
			}
		}
	}
}

var NodeUpdatableFields = []string{"name", "lastName", "mediaRef"}

// func HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
// 	id := r.PathValue("id")
// 	if update, err := bson.ReadDocument(r.Body); err != nil {
// 		log.Println("HandleUpdateNodeMongo error reading body:", err)
// 		w.WriteHeader(500)
// 	} else if err = UpdateByIdMongo(db.Nodes, id, update); err != nil {
// 		log.Println("HandleUpdateNodeMongo error updating:", err)
// 		w.WriteHeader(501)
// 	}
// }
