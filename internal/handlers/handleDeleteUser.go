package handlers

// import (
// 	"context"
// 	"net/http"

// 	"github.com/coldstar-507/node-server/internal/db"
// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/mongo"
// )

// func HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
// 	tag := r.PathValue("tag")
// 	ctx := context.Background()
// 	err := db.Mongo.UseSession(ctx, func(sc mongo.SessionContext) error {
// 		err := sc.StartTransaction()
// 		if err != nil {
// 			return err
// 		}

// 		db.Tags.DeleteOne(sc, bson.M{"_id": tag})
// 		_, err = db.Tags.InsertOne(sc, initValues["tag"])
// 		if err != nil {
// 			return err
// 		}

// 		_, err = db.Nodes.InsertOne(sc, initValues["node"])
// 		if err != nil {
// 			return err
// 		}

// 		_, err = db.Users.InsertOne(sc, initValues["user"])
// 		if err != nil {
// 			return err
// 		}

// 		return sc.CommitTransaction(context.Background())
// 	})

// }
