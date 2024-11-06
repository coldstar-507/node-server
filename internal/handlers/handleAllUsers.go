package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/coldstar-507/node-server/internal/db"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleAllUsers(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	cur, err := db.Users.Find(ctx, bson.D{})
	if err != nil {
		w.WriteHeader(500)
		log.Println("HandleAllNodes error finding:", err)
	}

	var m []map[string]any
	if err := cur.All(ctx, &m); err != nil {
		w.WriteHeader(501)
		log.Println("HandleAllNodes error cur.All:", err)
	}

	b, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		w.WriteHeader(502)
		log.Println("HandleAllNodes error marshaling:", err)
	}

	if _, err := w.Write(b); err != nil {
		w.WriteHeader(503)
		log.Println("HandleAllNodes error writing response:", err)
	}
}
