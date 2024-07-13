package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"log"
	"net/http"
	"strings"

	// "down4.com/utils"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgxutil"
	"github.com/vmihailenco/msgpack/v5"

	// "github.com/jackc/pgx/v5"
	// "github.com/jackc/pgxutil"
	// "github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/bson"
)

func GetMongoNodeById(id string) ([]byte, error) {
	raw, err := db.Nodes.FindOne(context.Background(), bson.M{"_id": id}).Raw()
	if err != nil {
		log.Printf("GetMongoNode error getting Raw() from FindOne() id=%s: %v\n", id, err)
		return nil, err
	}
	return raw, nil
}

func GetMongoNodesByIds(ids []string) ([]byte, error) {
	filter := bson.M{"_id": bson.M{"$in": ids}}
	cur, err := db.Nodes.Find(context.Background(), filter)
	if err != nil {
		log.Printf("GetMongoNodes error finding=%s: %v\n", strings.Join(ids, ","), err)
		return nil, err
	}

	buf := new(bytes.Buffer)
	for cur.Next(context.Background()) {
		binary.Write(buf, binary.BigEndian, uint16(len(cur.Current)))
		buf.Write(cur.Current)
	}
	return buf.Bytes(), nil
}

func GetMongoNodeByTag(tag string) ([]byte, error) {
	raw, err := db.Nodes.FindOne(context.Background(), bson.M{"tag": tag}).Raw()
	if err != nil {
		log.Printf("GetMongoNode error getting Raw() from FindOne() tag=%s: %v\n", tag, err)
		return nil, err
	}
	return raw, nil
}

func GetMongoNodesByTags(tags []string) ([]byte, error) {
	filter := bson.M{"tag": bson.M{"$in": tags}}
	cur, err := db.Nodes.Find(context.Background(), filter)
	if err != nil {
		log.Printf("GetMongoNodes error finding=%s: %v\n", strings.Join(tags, ","), err)
		return nil, err
	}

	buf := new(bytes.Buffer)
	for cur.Next(context.Background()) {
		binary.Write(buf, binary.BigEndian, uint16(len(cur.Current)))
		buf.Write(cur.Current)
	}
	return buf.Bytes(), nil
}

func GetMongoUserByTag(tag string) ([]byte, error) {
	raw, err := db.Users.FindOne(context.Background(), bson.M{"tag": tag}).Raw()
	if err != nil {
		log.Printf("GetUserByTag error getting Raw() from FindOne() tag=%s: %v\n", tag, err)
		return nil, err
	}
	return raw, nil
}

func GetMongoUserById(id string) ([]byte, error) {
	raw, err := db.Users.FindOne(context.Background(), bson.M{"_id": id}).Raw()
	if err != nil {
		log.Printf("GetUserById error getting Raw() from FindOne() id=%s: %v\n", id, err)
		return nil, err
	}
	return raw, nil
}

func GetUserById(id string) ([]byte, error) {
	sql, args := "SELECT * FROM users WHERE id = $1", []any{id}
	node, err := pgxutil.SelectRow(context.Background(), db.Pool, sql, args, pgx.RowToMap)
	if err != nil {
		return nil, err
	}
	packed, err := msgpack.Marshal(&node)
	if err != nil {
		return nil, err
	}
	return packed, nil
}

func GetUserByTag(tag string) ([]byte, error) {
	sql, args := "SELECT * FROM users WHERE tag = $1", []any{tag}
	node, err := pgxutil.SelectRow(context.Background(), db.Pool, sql, args, pgx.RowToMap)
	if err != nil {
		return nil, err
	}
	packed, err := msgpack.Marshal(&node)
	if err != nil {
		return nil, err
	}
	return packed, nil
}

func GetNodeById(id string) ([]byte, error) {
	sql, args := "SELECT * FROM nodes WHERE id = $1", []any{id}
	node, err := pgxutil.SelectRow(context.Background(), db.Pool, sql, args, pgx.RowToMap)
	if err != nil {
		return nil, err
	}
	packed, err := msgpack.Marshal(&node)
	if err != nil {
		return nil, err
	}
	return packed, nil
}

func GetNodeByTag(tag string) ([]byte, error) {
	sql, args := "SELECT * FROM nodes WHERE tag = $1", []any{tag}
	node, err := pgxutil.SelectRow(context.Background(), db.Pool, sql, args, pgx.RowToMap)
	if err != nil {
		return nil, err
	}
	packed, err := msgpack.Marshal(&node)
	if err != nil {
		return nil, err
	}
	return packed, nil
}

func GetNodesByIds(ids []string) ([]byte, error) {
	sql, args := "SELECT * FROM nodes WHERE id IN $1", []any{ids}
	nodes, err := pgxutil.Select(context.Background(), db.Pool, sql, args, pgx.RowToMap)
	if err != nil {
		return nil, err
	}
	packed, err := msgpack.Marshal(&nodes)
	if err != nil {
		return nil, err
	}
	return packed, nil
}

func GetNodesByTags(tags []string) ([]byte, error) {
	sql, args := "SELECT * FROM nodes WHERE tag IN $1", []any{tags}
	nodes, err := pgxutil.Select(context.Background(), db.Pool, sql, args, pgx.RowToMap)
	if err != nil {
		return nil, err
	}
	packed, err := msgpack.Marshal(&nodes)
	if err != nil {
		return nil, err
	}
	return packed, nil
}

func HandleGetNodesByTags(w http.ResponseWriter, r *http.Request) {
	tags := strings.Split(r.PathValue("tags"), ",")
	if nodes, err := GetMongoNodesByTags(tags); err != nil {
		w.WriteHeader(500)
		log.Println("HandleGetNodes error getting nodes:", err)
	} else if _, err = w.Write(nodes); err != nil {
		w.WriteHeader(501)
		log.Println("HandleGetNodes error writing nodes:", err)
	}
}

func HandleGetNodeByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	if node, err := GetMongoNodeByTag(tag); err != nil {
		w.WriteHeader(500)
		log.Println("HandleGetNode error getting node:", err)
	} else if _, err = w.Write(node); err != nil {
		w.WriteHeader(501)
		log.Println("HandleGetNode error writing response:", err)
	}
}

func HandleGetNodesByIds(w http.ResponseWriter, r *http.Request) {
	ids := strings.Split(r.PathValue("ids"), ",")
	if nodes, err := GetMongoNodesByIds(ids); err != nil {
		w.WriteHeader(500)
		log.Println("HandleGetNodes error getting nodes:", err)
	} else if _, err = w.Write(nodes); err != nil {
		w.WriteHeader(501)
		log.Println("HandleGetNodes error writing nodes:", err)
	}
}

func HandleGetNodeById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if node, err := GetMongoNodeById(id); err != nil {
		w.WriteHeader(500)
		log.Println("HandleGetNode error getting node:", err)
	} else if _, err = w.Write(node); err != nil {
		w.WriteHeader(501)
		log.Println("HandleGetNode error writing response:", err)
	}
}
