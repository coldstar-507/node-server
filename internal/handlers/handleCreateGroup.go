package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func AddToGroup(id string, ids []string) error {
	return db.Mongo.UseSession(context.Background(), func(sc mongo.SessionContext) error {
		err := sc.StartTransaction()
		if err != nil {
			return fmt.Errorf("error starting tx: %v", err)
		}

		addMems := bson.M{"$addToSet": bson.M{"members": bson.M{"$each": ids}}}
		_, err = db.Nodes.UpdateByID(sc, id, addMems)
		if err != nil {
			return fmt.Errorf("error updating members: %v", err)
		}

		filter := bson.M{"_id": bson.M{"$in": ids}}
		addConns := bson.M{"$addToSet": bson.M{"conns": id}}
		_, err = db.Nodes.UpdateMany(sc, filter, addConns)
		// _, err = db.Users.UpdateMany(sc, filter, addConns)
		if err != nil {
			return fmt.Errorf("error updating roots: %v", err)
		}

		err = sc.CommitTransaction(context.Background())
		if err != nil {
			return fmt.Errorf("error commiting tx: %v", err)
		}

		return nil
	})
}

func CreateGroup(group map[string]any) error {
	var panicErr error
	ctx := context.Background()
	err := db.Mongo.UseSession(ctx, func(sc mongo.SessionContext) error {
		err := sc.StartTransaction()

		if err != nil {
			return err
		}

		defer func() {
			// Handle panic and abort transaction if needed
			if r := recover(); r != nil {
				panicErr = fmt.Errorf("panic error: %v", r)
				log.Println(panicErr)
				sc.AbortTransaction(context.Background())
			}
		}()

		// node insertion
		_, err = db.Nodes.InsertOne(sc, group)
		if err != nil {
			return fmt.Errorf("error inserting node: %v", err)
		}

		// pushing root to members
		members := group["members"].([]interface{})
		filter := bson.M{"_id": bson.M{"$in": members}}
		groupId := group["_id"].(string)
		update := bson.M{"$addToSet": bson.M{"conns": groupId}}
		_, err = db.Nodes.UpdateMany(sc, filter, update)
		// _, err = db.Users.UpdateMany(sc, filter, update)
		if err != nil {
			return fmt.Errorf("error pushing conns: %v", err)
		}

		err = sc.CommitTransaction(context.Background())
		if err != nil {
			return fmt.Errorf("error commiting transaction: %v", err)
		}

		return nil
	})
	return errors.Join(err, panicErr)
}

func HandleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var vals map[string]any
	if err := msgpack.NewDecoder(r.Body).Decode(&vals); err != nil {
		log.Println("HandleCreateGroup error decoding group:", err)
		w.WriteHeader(500)
	} else if err = CreateGroup(vals); err != nil {
		log.Println("HandleCreateGroup error creating group:", err)
		w.WriteHeader(501)
	}
}

func HandleAddToGroup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ids := strings.Split(r.PathValue("ids"), ",")
	if err := AddToGroup(id, ids); err != nil {
		log.Println("HanldeAddToGroup:", err)
		w.WriteHeader(500)
	}
}
