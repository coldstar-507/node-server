package test0

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"time"

	"testing"

	db_listener "github.com/coldstar-507/node-server/internal"
	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"github.com/coldstar-507/utils"

	"go.mongodb.org/mongo-driver/bson"
)

// const jeff_id = "jeff_id"

// func deleteAll() {
// 	ctx := context.Background()
// 	db.Tags.DeleteMany(ctx, bson.D{})
// 	db.Nodes.DeleteMany(ctx, bson.D{})
// 	db.Users.DeleteMany(ctx, bson.D{})
// }

////////////////////////////////////////

func TestMain(m *testing.M) {
	db.InitMongo()
	defer db.ShutdownMongo()
	go db_listener.MongoUserListener()
	go db_listener.MongoNodeListener()
	<-time.NewTimer(time.Second * 2).C
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
			"_id":       "jeff_id",
			"tag":       "jeff",
			"name":      "Jeff",
			"type":      "user",
			"geohash":   "fjsd23j",
			"gender":    "male",
			"interests": []string{},
		},
		"user": map[string]any{
			"_id":    "jeff_id",
			"tag":    "jeff",
			"roots":  []string{},
			"images": []string{},
			"videos": []string{},
			"gifs":   []string{},
		},
	}

	if err := handlers.InitTransactionMongo(initValues); err != nil {
		t.Error(err)
	}
}

func TestPushRoot(t *testing.T) {
	root, userId := "test_root_0", "jeff_id"
	err := handlers.PushRoot(root, []string{userId})
	if err != nil {
		t.Error(err)
	}
	raw, _ := handlers.GetMongoUserById("jeff_id")
	t.Log(utils.SprettyPrint(raw))
}

func TestValidUsername(t *testing.T) {
	v0, v1 := handlers.ValidTagMongo("andrew"), handlers.ValidTagMongo("jeff")
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
		t.Log(utils.SprettyPrint(raw))
	}
}

func TestGetNodeById(t *testing.T) {
	if raw, err := handlers.GetMongoNodeById("jeff_id"); err != nil {
		t.Error(err)
	} else {
		t.Log(utils.SprettyPrint(raw))
	}
}

func TestPushMedias(t *testing.T) {
	err := handlers.PushMediasToId("jeff_id", "images", []string{"image1", "image2"})
	if err != nil {
		t.Error("TestPushMedias error pushing medias to jeff: ", err)
	}

	jeff, _ := handlers.GetMongoUserByTag("jeff")
	t.Log("jeff after PushMedias\n", utils.SprettyPrint(jeff))
}

func TestCreateGroup(t *testing.T) {
	group := map[string]any{
		"root": "group0_root",
		"node": map[string]any{
			"_id":          "group0_id",
			"tag":          "big_fellas",
			"name":         "Big Fellas",
			"type":         "group",
			"currentRoots": []string{"root0_group0_id"},
			"members":      []string{"jeff_id"},
		},
	}

	if err := handlers.CreateGroup(group); err != nil {
		t.Error(err)
	}

	jeff, _ := handlers.GetMongoUserByTag("jeff")
	t.Log("jeff after group0_id was created\n", utils.SprettyPrint(jeff))

}

func TestAddUsersToGroup(t *testing.T) {
	newMembers := []string{"jeff_id", "andrew_id", "scott_id", "helene_id", "david_id"}
	handlers.AddToGroup("group0_id", newMembers)
	abc, _ := handlers.GetMongoNodeByTag("big_fellas")
	t.Log("big_fellas after update\n", utils.SprettyPrint(abc))
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
		t.Log(utils.SprettyPrint(a))
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
		s, _ := json.MarshalIndent(a, "", "    ")
		t.Log(string(s))
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
		t.Log(utils.SprettyPrint(a))
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	tags := []string{"jeff", "big_fellas"}
	db.Tags.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": tags}})
	db.Nodes.DeleteMany(ctx, bson.M{"tag": bson.M{"$in": tags}})
	db.Users.DeleteMany(ctx, bson.M{"tag": bson.M{"$in": tags}})
}
