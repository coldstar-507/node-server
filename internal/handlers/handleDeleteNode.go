package handlers

import (
	"context"
	"net/http"

	"github.com/coldstar-507/node-server/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func DeleteNodeByTag(tag string) error {
	return db.Mongo.UseSession(context.Background(), func(sc mongo.SessionContext) error {
		if _, err := db.Nodes.DeleteOne(sc, bson.M{"tag": tag}); err != nil {
			return err
		} else if _, err = db.Tags.DeleteOne(sc, bson.M{"_id": tag}); err != nil {
			return err
		}
		return sc.CommitTransaction(context.Background())
	})
}

func HandleDeleteNodeByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	if len(tag) == 0 {
		w.WriteHeader(501)
	} else if err := DeleteNodeByTag(tag); err != nil {
		w.WriteHeader(500)
	}
}

func HandleDeleteNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dr, err := db.Nodes.DeleteOne(context.Background(), bson.M{"_id": id})
	if dr.DeletedCount != 0 && dr.DeletedCount != 1 && err != nil {
		w.WriteHeader(500)
	}
}
