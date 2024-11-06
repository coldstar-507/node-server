package handlers

import (
	"context"
	"errors"

	// "fmt"
	"log"
	"net/http"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgxutil"

	// "github.com/jackc/pgxutil"
	"github.com/vmihailenco/msgpack/v5"
	// "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// type initType struct {
// 	Id        string   `json:"id" msgpack:"id"`
// 	Neuter    string   `json:"neuter" msgpack:"neuter"`
// 	Secret    string   `json:"secret" msgpack:"secret"`
// 	Token     string   `json:"token" msgpack:"token"`
// 	Latitude  float64  `json:"latitude" msgpack:"latitude"`
// 	Longitude float64  `json:"longitude" msgpack:"longitude"`
// 	Geohash   string   `json:"geohash" msgpack:"geohash"`
// 	Interests []string `json:"interests" msgpack:"interests"`
// }

func InitTransactionMongo(initValues map[string]any) error {
	ctx := context.Background()
	return db.Mongo.UseSession(ctx, func(sc mongo.SessionContext) error {
		err := sc.StartTransaction()
		if err != nil {
			return err
		}
		_, err = db.Tags.InsertOne(sc, initValues["tag"])
		if err != nil {
			return err
		}

		_, err = db.Nodes.InsertOne(sc, initValues["node"])
		if err != nil {
			return err
		}

		_, err = db.Users.InsertOne(sc, initValues["user"])
		if err != nil {
			return err
		}

		return sc.CommitTransaction(context.Background())
	})
}

func InitTransactionMongo2(initValues bson.Raw) error {
	ctx := context.Background()
	return db.Mongo.UseSession(ctx, func(sc mongo.SessionContext) error {
		err := sc.StartTransaction()
		if err != nil {
			return err
		}

		tag := initValues.Lookup("node", "tag").StringValue()
		_, err = db.Tags.InsertOne(sc, bson.M{"_id": tag})
		if err != nil {
			return err
		}

		_, err = db.Nodes.InsertOne(sc, initValues)
		if err != nil {
			return err
		}

		return sc.CommitTransaction(context.Background())
	})
}

func InitTransaction(ctx context.Context, initValues map[string]any) error {
	return pgx.BeginTxFunc(ctx, db.Pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		userValues := initValues["user"].(map[string]any)
		nodeValues := initValues["node"].(map[string]any)
		err0 := pgxutil.InsertRow(ctx, tx, "users", userValues)
		err1 := pgxutil.InsertRow(ctx, tx, "nodes", nodeValues)
		return errors.Join(err0, err1)
	})
}

func HandleInit2(w http.ResponseWriter, r *http.Request) {
	if raw, err := bson.ReadDocument(r.Body); err != nil {
		log.Println("HandleInitMongo error reading init document:", err)
		w.WriteHeader(500)
	} else if err = InitTransactionMongo2(raw); err != nil {
		log.Println("HandleInitMongo error making init tx:", err)
		w.WriteHeader(501)
	}
}

// user pre-encode to bson? -> no, msgpack or json that shit
func HandleInit(w http.ResponseWriter, r *http.Request) {
	var initVals map[string]any
	if err := msgpack.NewDecoder(r.Body).Decode(&initVals); err != nil {
		log.Println("HandleInitMongo error reading init document:", err)
		w.WriteHeader(500)
	} else if err = InitTransactionMongo(initVals); err != nil {
		log.Println("HandleInitMongo error making init tx:", err)
		w.WriteHeader(501)
	}
}

// this doesn't work, cannot marshal bson.RawValue to bson document
// func HandleInit2(w http.ResponseWriter, r *http.Request) {
// 	if raw, err := bson.ReadDocument(r.Body); err != nil {
// 		log.Println("HandleInitMongo error reading init document:", err)
// 		w.WriteHeader(500)
// 	} else if err = InitTransactionMongo2(raw); err != nil {
// 		log.Println("HandleInitMongo error making init tx:", err)
// 		w.WriteHeader(501)
// 	}
// }

// func HandleInit(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()
// 	var initData map[string]any
// 	if err := msgpack.NewDecoder(r.Body).Decode(&initData); err != nil {
// 		w.WriteHeader(500)
// 		log.Printf("ERROR handlers.HandleInit decoding new user: %v\n", err)
// 		return
// 	}

// 	if err := pgxutil.InsertRow(ctx, db.Pool, "users", initData); err != nil {
// 		w.WriteHeader(501)
// 		log.Printf("ERROR handlers.HandleInit inserting new user: %v\n", err)
// 		return
// 	}
// }
