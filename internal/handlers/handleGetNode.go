package handlers

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgxutil"
	"github.com/vmihailenco/msgpack/v5"

	"go.mongodb.org/mongo-driver/bson"
)

func GetMongoNodeById(id string) ([]byte, error) {
	return db.Nodes.FindOne(context.Background(), bson.M{"_id": id}).Raw()

}

func GetMongoNodeByIdAfter(id string, lastUpdate int64) ([]byte, error) {
	filter := bson.M{"_id": id, "lastUpdate": bson.M{"$gt": lastUpdate}}
	return db.Nodes.FindOne(context.Background(), filter).Raw()
}

func GetMongoNodesByIds(ids []string, w io.Writer) error {
	filter := bson.M{"_id": bson.M{"$in": ids}}
	cur, err := db.Nodes.Find(context.Background(), filter)
	if err != nil {
		log.Printf("GetMongoNodes error finding=%s: %v\n", strings.Join(ids, ","), err)
		return err
	}

	for cur.Next(context.Background()) {
		l := uint16(len(cur.Current))
		if err = binary.Write(w, binary.BigEndian, l); err != nil {
			return err
		}
		if _, err = w.Write(cur.Current); err != nil {
			return err
		}
	}
	return nil
}

func GetMongoNodeByTag(tag string) ([]byte, error) {
	return db.Nodes.FindOne(context.Background(), bson.M{"tag": tag}).Raw()
}

func GetMongoNodesByTags(tags []string, w io.Writer) error {
	filter := bson.M{"tag": bson.M{"$in": tags}}
	cur, err := db.Nodes.Find(context.Background(), filter)
	if err != nil {
		log.Printf("GetMongoNodes error finding=%s: %v\n", strings.Join(tags, ","), err)
		return err
	}

	for cur.Next(context.Background()) {
		l := uint16(len(cur.Current))
		if err = binary.Write(w, binary.BigEndian, l); err != nil {
			return err
		}
		if _, err = w.Write(cur.Current); err != nil {
			return err
		}
	}
	return nil
}

func GetMongoUserByTag(tag string) ([]byte, error) {
	return db.Users.FindOne(context.Background(), bson.M{"tag": tag}).Raw()
}

func GetMongoUserById(id string) ([]byte, error) {
	return db.Users.FindOne(context.Background(), bson.M{"_id": id}).Raw()
}

func GetMongoUserByIdAfter(id string, lastUpdate int64) ([]byte, error) {
	filter := bson.M{"_id": id, "lastUpdate": bson.M{"$gt": lastUpdate}}
	return db.Users.FindOne(context.Background(), filter).Raw()
}

func HandleGetNodesByTags(w http.ResponseWriter, r *http.Request) {
	tags := strings.Split(r.PathValue("tags"), ",")
	if err := GetMongoNodesByTags(tags, w); err != nil {
		log.Println("HandleGetNodesByTags error:", err)
		w.WriteHeader(500)
	}
}

func HandleGetNodesByIds(w http.ResponseWriter, r *http.Request) {
	ids := strings.Split(r.PathValue("ids"), ",")
	if err := GetMongoNodesByIds(ids, w); err != nil {
		log.Println("HandleGetNodesByIds error getting nodes:", err)
		w.WriteHeader(500)
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

func HandleGetPrettyUserByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	var m map[string]any
	if user, err := GetMongoUserByTag(tag); err != nil {
		w.WriteHeader(500)
		log.Println("HandleGetPrettyUserByTag error getting user:", err)
	} else if err := bson.Unmarshal(user, &m); err != nil {
		w.WriteHeader(501)
		log.Println("HandleGetPrettyUserByTag error unmarshalling to map:", err)
	} else if b, err := json.MarshalIndent(m, "", "    "); err != nil {
		w.WriteHeader(502)
		log.Println("HandleGetPrettyUserByTag marshalling to json:", err)
	} else if _, err := w.Write(b); err != nil {
		w.WriteHeader(503)
		log.Println("HandleGetPrettyUserByTag writing response:", err)
	}
}

func HandleGetPrettyUserById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var m map[string]any
	if user, err := GetMongoUserById(id); err != nil {
		w.WriteHeader(500)
		log.Println("HandleGetPrettyUserById error getting user:", err)
	} else if err := bson.Unmarshal(user, &m); err != nil {
		w.WriteHeader(501)
		log.Println("HandleGetPrettyUserById error unmarshalling to map:", err)
	} else if b, err := json.MarshalIndent(m, "", "    "); err != nil {
		w.WriteHeader(502)
		log.Println("HandleGetPrettyUserById marshalling to json:", err)
	} else if _, err := w.Write(b); err != nil {
		w.WriteHeader(503)
		log.Println("HandleGetPrettyUserById writing response:", err)
	}
}

func HandleGetPrettyNodeByTag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	var m map[string]any
	if user, err := GetMongoNodeByTag(tag); err != nil {
		w.WriteHeader(500)
		log.Println("HandleGetPrettyNodeByTag error getting user:", err)
	} else if err := bson.Unmarshal(user, &m); err != nil {
		w.WriteHeader(501)
		log.Println("HandleGetPrettyNodeByTag error unmarshalling to map:", err)
	} else if b, err := json.MarshalIndent(m, "", "    "); err != nil {
		w.WriteHeader(502)
		log.Println("HandleGetPrettyNodeByTag marshalling to json:", err)
	} else if _, err := w.Write(b); err != nil {
		w.WriteHeader(503)
		log.Println("HandleGetPrettyNodeByTag writing response:", err)
	}
}

func HandleGetPrettyNodeById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var m map[string]any
	if user, err := GetMongoNodeById(id); err != nil {
		w.WriteHeader(500)
		log.Println("HandleGetPrettyNodeById error getting user:", err)
	} else if err := bson.Unmarshal(user, &m); err != nil {
		w.WriteHeader(501)
		log.Println("HandleGetPrettyNodeById error unmarshalling to map:", err)
	} else if b, err := json.MarshalIndent(m, "", "    "); err != nil {
		w.WriteHeader(502)
		log.Println("HandleGetPrettyNodeById marshalling to json:", err)
	} else if _, err := w.Write(b); err != nil {
		w.WriteHeader(503)
		log.Println("HandleGetPrettyNodeById writing response:", err)
	}
}

// sql

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
