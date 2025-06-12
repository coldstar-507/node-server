package handlers

import (
	"context"
	"net/http"

	"github.com/coldstar-507/node-server/internal/db"
	// "github.com/jackc/pgxutil"

	// "github.com/jackc/pgx/v5"
	// "github.com/jackc/pgxutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func ValidTagMongo(tag string) bool {
	ctx, filter := context.Background(), bson.M{"_id": tag}
	if s := db.Tags.FindOne(ctx, filter); s.Err() == mongo.ErrNoDocuments {
		return true
	} else {
		return false
	}
}

// func ValidUsernamePostgres(username string) bool {
// 	sql := "SELECT NOT EXISTS(SELECT 1 FROM users WHERE tag = $1)"
// 	valid, _ := pgxutil.SelectRow(context.Background(),
// 		db.Pool, sql, []any{username}, pgx.RowTo[bool])
// 	return valid
// }

func HandleValidTag(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("tag")
	if !ValidTagMongo(username) {
		w.WriteHeader(204)
	}
}

// func HandleValidUser(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()
// 	username := r.PathValue("username")
// 	sql := "select exists(select 1 from users where id = $1)"
// 	exists, err := pgxutil.SelectRow(ctx, db.Pool, sql, []any{username}, pgx.RowTo[bool])
// 	if err != nil {
// 		w.WriteHeader(500)
// 		log.Printf("ERROR handlers.ValidUser selecting existance of %s: %v\n", username, err)
// 		return
// 	}
// 	if exists {
// 		w.Write([]byte("false"))
// 	} else {
// 		w.Write([]byte("true"))
// 	}
// }
