package test

import (
	"context"
	"encoding/json"

	// "encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/coldstar-507/flatgen"
	"github.com/coldstar-507/node-server/internal/bsv"
	"github.com/coldstar-507/node-server/internal/db"
)

func TestMain(m *testing.M) {
	db.InitMongo()
	defer db.ShutdownMongo()
	code := m.Run()
	os.Exit(code)
}

// func TestDelete(t *testing.T) {
// 	ctx := context.Background()
// 	tags := []string{"jeff", "big_fellas"}
// 	db.Tags.DeleteMany(ctx, mapp{"_id": mapp{"$in": tags}})
// 	db.Nodes.DeleteMany(ctx, mapp{"tag": mapp{"$in": tags}})
// 	db.Users.DeleteMany(ctx, mapp{"tag": mapp{"$in": tags}})
// }

func fakeNode(tag string, age int, gender string, interests []string,
	lat, lon float64) map[string]any {
	id := tag + "_id"
	geoh := bsv.MakeGeohash(&flatgen.LatLonT{Lat: lat, Lon: lon})
	return map[string]any{
		"_id":        id,
		"type":       "user",
		"tag":        tag,
		"age":        age,
		"latitude":   lat,
		"longitude":  lon,
		"geohash":    geoh,
		"gender":     gender,
		"chatPlaces": []uint16{4000},
		"interests":  interests,
	}
}

type mapp = map[string]any

// func TestDepopulate(t *testing.T) {
// 	db.Nodes.DeleteMany(context.Background(),
// 		mapp{"tag": mapp{"$in": []string{
// 			"alex", "scott", "andrew", "helene", "david", // "jeff",
// 		}}},
// 	)
// }

// func TestPopulate(t *testing.T) {
// 	// we already use jeff in another test, so it creates conflicts
// 	// fakeUsr("jeff", 20, "male", []string{"creatine", "money"}, 48.43202, -68.50756),
// 	docs := []any{
// 		fakeNode("alex", 50, "male", []string{"freedom", "food"}, 29.782209, -95.35466),
// 		fakeNode("scott", 26, "male", []string{"money", "coffee"}, 46.82421, -71.22137),
// 		fakeNode("andrew", 29, "male", []string{"kush", "creatine"}, 46.8242, -71.2213),
// 		fakeNode("helene", 58, "female", []string{"coffee"}, 46.82421, -71.22137),
// 		fakeNode("david", 57, "male", []string{"sugar", "coffee"}, 46.82421, -71.22137),
// 	}

// 	if _, err := db.Nodes.InsertMany(context.Background(), docs); err != nil {
// 		t.Error(err)
// 	}
// }

func TestListNodes(t *testing.T) {
	cur, err := db.Nodes.Find(context.Background(), mapp{})
	if err != nil {
		t.Error(err)
	}

	var all []mapp
	if err := cur.All(context.Background(), &all); err != nil {
		t.Error(err)
	}

	b, err := json.MarshalIndent(&all, "", "    ")
	if err != nil {
		t.Error(err)
	}

	fmt.Println(string(b))
}

// initValues := map[string]any{
// 	"tag": map[string]any{
// 		"_id": tag,
// 	},
// 	"node": map[string]any{
// 		"_id":  id,
// 		"tag":  tag,
// 		"name": fn,
// 		"type": "user",
// 	},
// 	"user": map[string]any{
// 		"_id":       id,
// 		"tag":       tag,
// 		"age":       age,
// 		"latitude":  lat,
// 		"longitude": lon,
// 		"geohash":   geoh,
// 		"gender":    gender,
// 		"roots":     []string{},
// 		"images":    []string{},
// 		"videos":    []string{},
// 		"gifs":      []string{},
// 		"interests": interests,
// 	},
// }
