package handlers

import (
	"context"
	// "fmt"
	// "log"
	// "net/http"

	// "strings"

	// "github.com/coldstar-507/node-server/internal/db"
	// "github.com/coldstar-507/utils/utils"
	// "github.com/vmihailenco/msgpack/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func AddToSetMongo(coll *mongo.Collection,
	field string, values []string, targets []string) error {

	filter := bson.M{"_id": bson.M{"$in": targets}}
	update := bson.M{"$addToSet": bson.M{field: bson.M{"$each": values}}}
	_, err := coll.UpdateMany(context.Background(), filter, update)
	return err
}

// func PushToSet(table string, field string, values []string, targets []string) error {
// 	targets_ := utils.Map(targets, func(s string) string { return "'" + s + "'" })
// 	sql := fmt.Sprintf(`
//         UPDATE %s
//         SET %s = (SELECT ARRAY(SELECT DISTINCT UNNEST(%s || $1)))
//         WHERE id IN (%s);`, table, field, field, strings.Join(targets_, ","))
// 	_, err := db.Pool.Exec(context.Background(), sql, values)
// 	return err
// }

// func PushMediasToId(id, mediaType string, medias []string) error {
// 	filter := bson.M{"_id": id}
// 	update := bson.M{"$addToSet": bson.M{mediaType: bson.M{"$each": medias}}}
// 	_, err := db.Users.UpdateMany(context.Background(), filter, update)
// 	return err
// }

// func PushMediasToId2(id, mediaType string, medias []string) error {
// 	filter := bson.M{"_id": id}
// 	updateKey := "user." + mediaType
// 	update := bson.M{"$addToSet": bson.M{updateKey: bson.M{"$each": medias}}}
// 	_, err := db.Nodes.UpdateMany(context.Background(), filter, update)
// 	return err
// }

// func PushRoot(root string, userIds []string) error {
// 	filter := bson.M{"_id": bson.M{"$in": userIds}}
// 	update := bson.M{"$addToSet": bson.M{"roots": root}}
// 	_, err := db.Users.UpdateMany(context.Background(), filter, update)
// 	return err
// }

// func HandlePushRoot(w http.ResponseWriter, r *http.Request) {
// 	root := r.PathValue("root")
// 	userIds := strings.Split(r.PathValue("ids"), ",")
// 	err := PushRoot(root, userIds)
// 	if err != nil {
// 		log.Println("HandlePushRoot error:", err)
// 		w.WriteHeader(500)
// 	}
// }

// var updatableUserKeys = []string{"phone"}

// func HandleUpdateUserById(w http.ResponseWriter, r *http.Request) {
// 	id := r.PathValue("id")
// 	var update map[string]any
// 	if err := msgpack.NewDecoder(r.Body).Decode(&update); err != nil {
// 		log.Println("HandleUpdateUser: error decoding update:", err)
// 		w.WriteHeader(500)
// 	} else {
// 		for k := range update {
// 			if !utils.Contains(k, updatableUserKeys) {
// 				delete(update, k)
// 			}
// 		}

// 		if len(update) > 0 {
// 			update_ := map[string]any{"$set": update}
// 			if err = UpdateByIdMongo(db.Users, id, update_); err != nil {
// 				log.Println("HandleUpdateUserById: error updating:", err)
// 				w.WriteHeader(501)
// 			}
// 		}
// 	}
// }

// func HandlePushMedias(w http.ResponseWriter, r *http.Request) {
// 	id := r.PathValue("id")
// 	medias := strings.Split(r.PathValue("medias"), ",")
// 	mediaType := r.PathValue("type")
// 	if err := PushMediasToId(id, mediaType, medias); err != nil {
// 		log.Println("HandlePushMedias error:", err)
// 		w.WriteHeader(500)
// 	}
// }

// func HandlePushNfts(w http.ResponseWriter, r *http.Request) {
// 	id := r.PathValue("id")
// 	ids := strings.Split(r.PathValue("ids"), ",")
// 	if len(ids) < 1 {
// 		log.Println("HandlePushNfts: no ids")
// 		return
// 	}

// 	update := bson.M{"$addToSet": bson.M{"nfts": bson.M{"$each": ids}}}
// 	_, err := db.Users.UpdateByID(context.Background(), id, update)
// 	if err != nil {
// 		log.Println("HandlePushNfts error:", err)
// 		w.WriteHeader(500)
// 	}
// }

// func HandlePushMedias2(w http.ResponseWriter, r *http.Request) {
// 	id := r.PathValue("id")
// 	medias := strings.Split(r.PathValue("medias"), ",")
// 	mediaType := r.PathValue("type")
// 	if err := PushMediasToId2(id, mediaType, medias); err != nil {
// 		log.Println("HandlePushMedias error:", err)
// 		w.WriteHeader(500)
// 	}
// }
