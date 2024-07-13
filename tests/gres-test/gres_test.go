package test0

import (
	"context"
	"os"

	"testing"

	"github.com/coldstar-507/node-server/internal/db"
	"github.com/coldstar-507/node-server/internal/handlers"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgxutil"
	"github.com/vmihailenco/msgpack/v5"
)

const jeff_id = "jeff_id"

// func deleteAll() {
// 	ctx := context.Background()
// 	db.Tags.DeleteMany(ctx, bson.D{})
// 	db.Nodes.DeleteMany(ctx, bson.D{})
// 	db.Users.DeleteMany(ctx, bson.D{})
// }

func deleteJeffGres() {
	db.Pool.Exec(context.Background(), `
		DELETE FROM nodes WHERE tag IN ('jeff', 'big_fellas');
		DELETE FROM users WHERE tag = 'jeff';`)
}

////////////////////////////////////////

func TestMain(m *testing.M) {
	db.Init()
	// db.Pool.Exec(context.Background(), "DROP TABLE nodes;")
	/// db.Pool.Exec(context.Background(), "DROP TABLE users;")
	deleteJeffGres()
	defer db.ShutDown()
	code := m.Run()
	os.Exit(code)
}

func TestInitGres(t *testing.T) {
	initValues := map[string]any{
		"node": map[string]any{
			"id":   jeff_id,
			"tag":  "jeff",
			"name": "Jeff",
			"type": "user",
		},
		"user": map[string]any{
			"id":        jeff_id,
			"tag":       "jeff",
			"geohash":   "fjsd23j",
			"token":     "a-token",
			"gender":    "male",
			"neuter":    "a-derivable-key",
			"roots":     []string{},
			"images":    []string{},
			"videos":    []string{},
			"gifs":      []string{},
			"interests": []string{},
		},
	}
	if err := handlers.InitTransaction(context.Background(), initValues); err != nil {
		t.Error(err)
	}
}

func TestValidUsernameGres(t *testing.T) {
	v0 := handlers.ValidUsernamePostgres("andrew")
	v1 := handlers.ValidUsernamePostgres("jeff")
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

func TestGetNodeByTagGres(t *testing.T) {
	if raw, err := handlers.GetNodeByTag("jeff"); err != nil {
		t.Error(err)
	} else {
		var m map[string]any
		msgpack.Unmarshal(raw, &m)
		t.Log(m)
	}
}

func TestGetNodeByIdGres(t *testing.T) {
	if raw, err := handlers.GetNodeById(jeff_id); err != nil {
		t.Error(err)
	} else {
		var m map[string]any
		msgpack.Unmarshal(raw, &m)
		t.Log(m)
	}
}

func TestPushChatRootGres(t *testing.T) {
	err := handlers.PushToSet("users", "roots", []string{"andrew_id"}, []string{jeff_id})
	if err != nil {
		t.Error(err)
	}
	var m map[string]any
	jeff, _ := handlers.GetUserByTag("jeff")
	msgpack.Unmarshal(jeff, &m)
	t.Log("jeff after PushChatRoot", m)
}

func TestPushMediasGres(t *testing.T) {
	values, targets := []string{"image1", "image2", "image3"}, []string{jeff_id}
	if err := handlers.PushToSet("users", "images", values, targets); err != nil {
		t.Error(err)
	}
	var m map[string]any
	jeff, _ := handlers.GetUserByTag("jeff")
	msgpack.Unmarshal(jeff, &m)
	t.Log("jeff after PushMedias", m)
}

func TestUpdateNodeByIdGres(t *testing.T) {
	update := map[string]any{"lastName": "Harrisson"}
	if err := handlers.UpdateNodeById(jeff_id, update); err != nil {
		t.Error("TestUpdateNode error updating jeff: ", err)
	}
	var m map[string]any
	jeff, _ := handlers.GetNodeByTag("jeff")
	msgpack.Unmarshal(jeff, &m)
	t.Log("jeff after update", m)
}

func TestUploadNodeGres(t *testing.T) {
	node := map[string]any{
		"id":      "group0_id",
		"tag":     "big_fellas",
		"name":    "Big Fellas",
		"type":    "group",
		"members": []string{jeff_id, "andrew_id"},
	}
	if err := handlers.UploadNode(node); err != nil {
		t.Error(err)
	}
}

func TestAddUsersToGroupGres(t *testing.T) {
	newMembers := []string{jeff_id, "scott_id", "helene_id", "david_id"}
	err := handlers.PushToSet("nodes", "members", newMembers, []string{"group0_id"})
	if err != nil {
		t.Error("TestAddUsersToGroupGres error:", err)
	}
	var m map[string]any
	abc, _ := handlers.GetNodeByTag("big_fellas")
	msgpack.Unmarshal(abc, &m)
	t.Log("big_fellas after update", m)
}

// ////////////////////////////////
func TestBasicReadAllNodesGres(t *testing.T) {
	t.Log("Reading all nodes")
	vals, err := pgxutil.Select(context.Background(),
		db.Pool, "SELECT * FROM nodes", nil, pgx.RowToMap)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(vals)
	}
}

func TestBasicReadAllUsersGres(t *testing.T) {
	t.Log("Reading all users")
	vals, err := pgxutil.Select(context.Background(),
		db.Pool, "SELECT * FROM users", nil, pgx.RowToMap)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(vals)
	}
}
