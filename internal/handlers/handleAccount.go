package handlers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/coldstar-507/node-server/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func hashStr(str string) string {
	h := sha1.New()
	h.Write([]byte(str))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

var ErrAccountAlreadyExists = errors.New("account already exists")

func DeleteAllAccounts() {
	db.Accounts.DeleteMany(context.Background(), bson.D{})
}

func HandleAllAccounts(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	res, err := db.Accounts.Find(ctx, bson.D{})
	var m []map[string]any
	if err != nil {
		w.WriteHeader(500)
	} else if err = res.All(ctx, &m); err != nil {
		w.WriteHeader(501)
	} else if b, err := json.MarshalIndent(m, "", "    "); err != nil {
		w.WriteHeader(502)
	} else if _, err = w.Write(b); err != nil {
		w.WriteHeader(503)
	}
}

func HandleAccountState(w http.ResponseWriter, r *http.Request) {
	phone := r.PathValue("phone")
	nodeId := r.PathValue("nodeId")
	filter := bson.D{{Key: "phone", Value: phone}}
	raw, err := db.Accounts.FindOne(context.Background(), filter).Raw()
	if err == mongo.ErrNoDocuments {
		w.WriteHeader(201)
	} else if err != nil {
		w.WriteHeader(500)
	} else if raw.Lookup("nodeId").StringValue() == nodeId {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(501)
	}
}

func HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	phone := r.PathValue("phone")
	countryCode := r.PathValue("countryCode")
	nodeId := r.PathValue("nodeId")
	err := createAccount(ctx, phone, countryCode, nodeId)
	if err == ErrAccountAlreadyExists {
		w.WriteHeader(500)
	} else if err != nil {
		w.WriteHeader(501)
	}
}

func createAccount(ctx context.Context, phone, countryCode, nodeId string) error {
	txPref := options.Transaction().SetReadPreference(readpref.Primary())
	return db.Mongo.UseSession(ctx, func(sc mongo.SessionContext) error {
		err := sc.StartTransaction(txPref)
		if err != nil {
			fmt.Println("createAccount: err starting tx:", err)
			return err
		}
		filter := bson.M{"phone": phone}
		err = db.Accounts.FindOne(sc, filter).Err()
		if err != mongo.ErrNoDocuments {
			fmt.Println("createAccount: err findOne:", err)
			return ErrAccountAlreadyExists
		}

		_, err = db.Accounts.InsertOne(sc, bson.M{
			"_id":    hashStr(phone + nodeId),
			"phone":  phone,
			"nodeId": nodeId,
		})
		if err != nil {
			fmt.Println("createAccount: err insertOne:", err)
			return err
		}

		_, err = db.Nodes.UpdateByID(sc, nodeId, bson.M{
			"$set": bson.M{
				"phone":       phone,
				"countryCode": countryCode,
				"verified":    true,
			},
			"$currentDate": bson.M{
				"lastUpdate": true,
			},
		})
		if err != nil {
			fmt.Println("createAccount: err updateById:", err)
			return err
		}
		return sc.CommitTransaction(ctx)
	})
}
