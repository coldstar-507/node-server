package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	// "log"
	// "net/http"
	"strings"

	// "down4.com/utils"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	// "go.mongodb.org/mongo-driver/bson/primitive"
)

// func HandlePushChatRoot(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()
// 	root := r.PathValue("root")
// 	mapFunc := func(s string) primitive.ObjectID { return utils.MakeMongoId_(s) }
// 	targets := utils.Map(strings.Split(r.PathValue("targets"), ","), mapFunc)
// 	filter := bson.M{"_id": bson.M{"$in": targets}}
// 	update := bson.M{"$addToSet": bson.M{"private.roots": root}}
// 	if _, err := db.Nodes.UpdateMany(ctx, filter, update); err != nil {
// 		log.Println("HandlePushChatRoot error: ", err)
// 		w.WriteHeader(500)
// 		w.Write([]byte(err.Error()))
// 	}
// }

// func HandlePushMedias(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()
// 	medias := strings.Split(r.PathValue("medias"), ",")
// 	_id := utils.MakeMongoId_(r.PathValue("root"))
// 	filter := bson.M{"_id": _id}
// 	update := bson.M{"$addToSet": bson.M{"private.medias": medias}}
// 	if _, err := db.Nodes.UpdateMany(ctx, filter, update); err != nil {
// 		log.Println("HandlePushMedias error: ", err)
// 		w.WriteHeader(500)
// 		w.Write([]byte(err.Error()))
// 	}
// }

func AddToSetMongo(coll *mongo.Collection, field string, values []string, targets []string) error {
	filter := bson.M{"_id": bson.M{"$in": targets}}
	update := bson.M{"$addToSet": bson.M{field: bson.M{"$each": values}}}
	_, err := coll.UpdateMany(context.Background(), filter, update)
	return err
}

// func PushChatIdMongo(id string, targets []string) error {
// 	return AddToSetMongo(db.Users, "roots", []string{id}, targets)
// 	// filter := bson.M{"_id": bson.M{"$in": targets}}
// 	// update := bson.M{"$addToSet": bson.M{"chats": id}}
// 	// _, err := db.Users.UpdateMany(context.Background(), filter, update)
// 	// return err
// }

// func PushChatId(id string, targets []string) error {
// 	sql := `UPDATE users
//                 SET chats = CASE WHEN NOT ($1 = ANY(chats)) array_append(chats, $1)
//                             ELSE chats END WHERE id IN $2`
// 	_, err := db.Pool.Exec(context.Background(), sql, id, targets)
// 	return err
// }

func PushToSet(table string, field string, values []string, targets []string) error {
	targets_ := utils.Map(targets, func(s string) string { return "'" + s + "'" })
	sql := fmt.Sprintf(`
        UPDATE %s
        SET %s = (SELECT ARRAY(SELECT DISTINCT UNNEST(%s || $1)))
        WHERE id IN (%s);`, table, field, field, strings.Join(targets_, ","))
	_, err := db.Pool.Exec(context.Background(), sql, values)
	return err
}

// func PushMedias(id string, medias []string) error {
// 	sql := `UPDATE users
//                 SET medias = (SELECT ARRAY(SELECT DISTINCT UNNEST(medias || $2)))
//                 WHERE id = $1`
// 	_, err := db.Pool.Exec(context.Background(), sql, id, medias)
// 	return err
// }

// func HandlePushChatId(w http.ResponseWriter, r *http.Request) {
// 	id := r.PathValue("id")
// 	targets := utils.MakeMongoIds(strings.Split(r.PathValue("targets"), ","))
// 	if err := PushChatIdMongo(id, targets); err != nil {
// 		log.Println("HandlePushChatTag error: ", err)
// 		w.WriteHeader(500)
// 		w.Write([]byte(err.Error()))
// 	}
// }

func PushMediasToId(id, mediaType string, medias []string) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$addToSet": bson.M{mediaType: bson.M{"$each": medias}}}
	_, err := db.Users.UpdateMany(context.Background(), filter, update)
	return err
}

func PushMediasToId2(id, mediaType string, medias []string) error {
	filter := bson.M{"_id": id}
	updateKey := "user." + mediaType
	update := bson.M{"$addToSet": bson.M{updateKey: bson.M{"$each": medias}}}
	_, err := db.Nodes.UpdateMany(context.Background(), filter, update)
	return err
}

func PushRoot(root string, userIds []string) error {
	filter := bson.M{"_id": bson.M{"$in": userIds}}
	update := bson.M{"$addToSet": bson.M{"roots": root}}
	_, err := db.Users.UpdateMany(context.Background(), filter, update)
	return err
}

func HandlePushRoot(w http.ResponseWriter, r *http.Request) {
	root := r.PathValue("root")
	userIds := strings.Split(r.PathValue("ids"), ",")
	err := PushRoot(root, userIds)
	if err != nil {
		log.Println("HandlePushRoot error:", err)
		w.WriteHeader(500)
	}
}

func HandleUpdateInterests(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	interests := strings.Split(r.PathValue("interests"), ",")
	update := bson.M{"$set": bson.M{"interests": interests}}
	_, err := db.Nodes.UpdateByID(context.Background(), id, update)
	if err != nil {
		log.Println("HandleUpdateInterests error:", err)
		w.WriteHeader(500)
	}
}

func HandlePushMedias(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	medias := strings.Split(r.PathValue("medias"), ",")
	mediaType := r.PathValue("type")
	if err := PushMediasToId(id, mediaType, medias); err != nil {
		log.Println("HandlePushMedias error:", err)
		w.WriteHeader(500)
	}
}

func HandlePushNfts(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ids := strings.Split(r.PathValue("ids"), ",")
	if len(ids) < 1 {
		log.Println("HandlePushNfts: no ids")
		return
	}

	update := bson.M{"$addToSet": bson.M{"nfts": bson.M{"$each": ids}}}
	_, err := db.Users.UpdateByID(context.Background(), id, update)
	if err != nil {
		log.Println("HandlePushNfts error:", err)
		w.WriteHeader(500)
	}
}

func HandlePushMedias2(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	medias := strings.Split(r.PathValue("medias"), ",")
	mediaType := r.PathValue("type")
	if err := PushMediasToId2(id, mediaType, medias); err != nil {
		log.Println("HandlePushMedias error:", err)
		w.WriteHeader(500)
	}
}
