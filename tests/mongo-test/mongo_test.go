package test0

import (
	"context"
	"net/http/httptest"
	"os"

	"testing"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"

	"go.mongodb.org/mongo-driver/bson"
)

const jeff_id = "jeff_id"

// func deleteAll() {
// 	ctx := context.Background()
// 	db.Tags.DeleteMany(ctx, bson.D{})
// 	db.Nodes.DeleteMany(ctx, bson.D{})
// 	db.Users.DeleteMany(ctx, bson.D{})
// }

func deleteJeff() {
	ctx := context.Background()
	db.Tags.DeleteMany(ctx, bson.M{"_id": "jeff"})
	db.Nodes.DeleteMany(ctx, bson.M{"tag": bson.M{"$in": []string{"jeff", "big_fellas"}}})
	db.Users.DeleteMany(ctx, bson.M{"tag": "jeff"})
}

////////////////////////////////////////

func TestMain(m *testing.M) {
	db.InitMongo()
	// deleteAll()
	deleteJeff()
	defer db.ShutdownMongo()
	code := m.Run()
	os.Exit(code)
}

func TestPing(t *testing.T) {
	req := httptest.NewRequest("GET", "/ping", nil)
	res := httptest.NewRecorder()
	handlers.HandlePing(res, req)
	if res.Code != 200 {
		t.Error("TestPing error")
	} else {
		t.Log(res.Body.String())
	}
}

func TestInit(t *testing.T) {
	initValues := map[string]any{
		"tag": map[string]any{
			"_id": "jeff",
		},
		"node": map[string]any{
			"_id":  jeff_id,
			"tag":  "jeff",
			"name": "Jeff",
			"type": "user",
		},
		"user": map[string]any{
			"_id":       jeff_id,
			"tag":       "jeff",
			"geohash":   "fjsd23j",
			"gender":    "male",
			"roots":     []string{},
			"medias":    []string{},
			"interests": []string{},
		},
	}
	if err := handlers.InitTransactionMongo(initValues); err != nil {
		t.Error(err)
	}
}

func TestValidUsername(t *testing.T) {
	v0, v1 := handlers.ValidUsernameMongo("andrew"), handlers.ValidUsernameMongo("jeff")
	if !v0 {
		t.Log("andrew should be a valid username")
	}
	if v1 {
		t.Log("jeff should be an invalid username")
	}
	if !v0 || v1 {
		t.Fail()
	}
}

func TestGetNodeByTag(t *testing.T) {
	if raw, err := handlers.GetMongoNodeByTag("jeff"); err != nil {
		t.Error(err)
	} else {
		var m map[string]any
		bson.Unmarshal(raw, &m)
		t.Log(m)
	}
}

func TestGetNodeById(t *testing.T) {
	if raw, err := handlers.GetMongoNodeById(jeff_id); err != nil {
		t.Error(err)
	} else {
		var m map[string]any
		bson.Unmarshal(raw, &m)
		t.Log(m)
	}
}

func TestPushChatRoot(t *testing.T) {
	update, _ := bson.Marshal(bson.M{"$addToSet": bson.M{"roots": "andrew_id"}})
	if err := handlers.UpdateByIdMongo(db.Users, jeff_id, update); err != nil {
		t.Error("TestPushChatRoot error pushing andrew to jeff: ", err)
	}
	var m map[string]any
	jeff, _ := handlers.GetMongoUserByTag("jeff")
	bson.Unmarshal(jeff, &m)
	t.Log("jeff after PushChatRoot", m)
}

func TestPushMedias(t *testing.T) {
	update, _ := bson.Marshal(
		bson.M{"$addToSet": bson.M{"images": bson.M{"$each": []string{"image1", "image2"}}}},
	)
	if err := handlers.UpdateByIdMongo(db.Users, jeff_id, update); err != nil {
		t.Error("TestPushMedias error pushing medias to jeff: ", err)
	}
	var m map[string]any
	jeff, _ := handlers.GetMongoUserByTag("jeff")
	bson.Unmarshal(jeff, &m)
	t.Log("jeff after PushMedias", m)
}

func TestUpdateNodeById(t *testing.T) {
	doc := bson.M{"$set": bson.M{"lastName": "Harrisson", "mediaId": "lol_id"}}
	update, _ := bson.Marshal(doc)
	if err := handlers.UpdateByIdMongo(db.Nodes, jeff_id, update); err != nil {
		t.Error("TestUpdateNode error updating jeff: ", err)
	}
	var m map[string]any
	jeff, _ := handlers.GetMongoNodeByTag("jeff")
	bson.Unmarshal(jeff, &m)
	t.Log("jeff after update", m)
}

func TestUploadNode(t *testing.T) {
	node, _ := bson.Marshal(bson.M{
		"_id":     "group0_id",
		"tag":     "big_fellas",
		"name":    "Big Fellas",
		"type":    "group",
		"members": []string{jeff_id, "andrew_id"},
	})
	if err := handlers.UploadNodeMongo(node); err != nil {
		t.Error(err)
	}
}

func TestAddUsersToGroup(t *testing.T) {
	newMembers := []string{jeff_id, "scott_id", "helene_id", "david_id"}
	doc := bson.M{"$addToSet": bson.M{"members": bson.M{"$each": newMembers}}}
	update, _ := bson.Marshal(doc)
	if err := handlers.UpdateByIdMongo(db.Nodes, "group0_id", update); err != nil {
		t.Error("TestAddUsersToGroup error:", err)
	}
	var m map[string]any
	abc, _ := handlers.GetMongoNodeByTag("big_fellas")
	bson.Unmarshal(abc, &m)
	t.Log("big_fellas after update", m)
}

// ////////////////////////////////
func TestBasicReadAllNodes(t *testing.T) {
	t.Log("Reading all nodes")
	cur, err := db.Nodes.Find(context.Background(), bson.D{})
	if err != nil {
		t.Error(err)
	} else {
		var a []map[string]any
		cur.All(context.Background(), &a)
		t.Log(a)
	}
}

func TestBasicReadAllUsers(t *testing.T) {
	t.Log("Reading all users")
	cur, err := db.Users.Find(context.Background(), bson.D{})
	if err != nil {
		t.Error(err)
	} else {
		var a []map[string]any
		cur.All(context.Background(), &a)
		t.Log(a)
	}
}

func TestBasicReadAllTags(t *testing.T) {
	t.Log("Reading all tags")
	cur, err := db.Tags.Find(context.Background(), bson.D{})
	if err != nil {
		t.Error(err)
	} else {
		var a []map[string]any
		cur.All(context.Background(), &a)
		t.Log(a)
	}
}
