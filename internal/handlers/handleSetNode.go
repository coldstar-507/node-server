package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/jackc/pgxutil"
	// "github.com/jackc/pgxutil"
	// "github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/bson"
)

func UploadNode(node map[string]any) error {
	return pgxutil.InsertRow(context.Background(), db.Pool, "nodes", node)
}

func UploadNodeMongo(node bson.Raw) error {
	_, err := db.Nodes.InsertOne(context.Background(), node)
	return err
}

func HandleUploadNode(w http.ResponseWriter, r *http.Request) {
	if bson, err := bson.ReadDocument(r.Body); err != nil {
		log.Println("HandleUploadNodeMongo error reading body:", err)
		w.WriteHeader(500)
	} else if _, err = db.Nodes.InsertOne(context.Background(), bson); err != nil {
		log.Println("HandleUploadNodeMongo error inserting node:", err)
		w.WriteHeader(501)
	}
}

// func HandleUploadNode(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()
// 	var m map[string]any
// 	if err := msgpack.NewDecoder(r.Body).Decode(&m); err != nil {
// 		w.WriteHeader(500)
// 		log.Println("HandleUploadNode error unmarshalling msgpack:", err)
// 		return
// 	}
// 	if err := pgxutil.InsertRow(ctx, db.Pool, "nodes", m); err != nil {
// 		w.WriteHeader(500)
// 		log.Println("HandleUploadNode error inserting node:", err)
// 		return
// 	}
// }
