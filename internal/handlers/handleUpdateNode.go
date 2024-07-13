package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/jackc/pgxutil"
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

func UpdateByIdMongo(coll *mongo.Collection, id string, update bson.Raw) error {
	_, err := coll.UpdateByID(context.Background(), id, update)
	return err
}

func HandleUpdateNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if update, err := bson.ReadDocument(r.Body); err != nil {
		log.Println("HandleUpdateNodeMongo error reading body:", err)
		w.WriteHeader(500)
	} else if err = UpdateByIdMongo(db.Nodes, id, update); err != nil {
		log.Println("HandleUpdateNodeMongo error updating:", err)
		w.WriteHeader(501)
	}
}

func HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if update, err := bson.ReadDocument(r.Body); err != nil {
		log.Println("HandleUpdateNodeMongo error reading body:", err)
		w.WriteHeader(500)
	} else if err = UpdateByIdMongo(db.Nodes, id, update); err != nil {
		log.Println("HandleUpdateNodeMongo error updating:", err)
		w.WriteHeader(501)
	}
}
