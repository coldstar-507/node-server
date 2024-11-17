package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/coldstar-507/node-server/internal/bsv"
	"github.com/coldstar-507/node-server/internal/db"
)

func TestMain(m *testing.M) {
	db.InitMongo()
	defer db.ShutdownMongo()
	dist := bsv.GeoDist(
		bsv.LatLon{Lat: 46.31585857259396, Lon: -72.5494256605008},
		bsv.LatLon{Lat: 46.77383958951538, Lon: -71.09242612644653},
	)
	dist2 := bsv.GeoDist(
		bsv.LatLon{Lat: 46.94589131490488, Lon: -71.2888815994351},
		bsv.LatLon{Lat: 46.807092314421034, Lon: -71.22967468461174},
	)
	fmt.Println("bsv test main")
	fmt.Println("dist:", dist)
	fmt.Println("dist2:", dist2)
	code := m.Run()
	os.Exit(code)
}

func TestLayerQuebecCity(t *testing.T) {
	a := &bsv.Area{
		Center: bsv.LatLon{Lat: 46.82749877182937, Lon: -71.22776575143017},
		Perim: []bsv.LatLon{
			{Lat: 46.90136539189456, Lon: -71.22733231706924, RefDist: 16.076},
			{Lat: 46.80923104124465, Lon: -71.0691661771562, RefDist: 16.076},
			{Lat: 46.7307167054483, Lon: -71.26152704981182, RefDist: 16.076},
			{Lat: 46.824275505621095, Lon: -71.43072610195287, RefDist: 16.076},
		},
	}
	layers := bsv.CalcLayers(a)
	fmt.Println("quebec city layers:\n", layers)
}

func TestLayerQuebecCities(t *testing.T) {
	a := &bsv.Area{
		Center: bsv.LatLon{Lat: 46.36683529877258, Lon: -72.58105744075569},
		Perim: []bsv.LatLon{
			{Lat: 44.9523555993882, Lon: -72.40506027364277, RefDist: 122.51},
			{Lat: 46.198697681289964, Lon: -74.10291529755564, RefDist: 122.51},
			{Lat: 47.23240565782047, Lon: -72.5834542471733, RefDist: 122.51},
			{Lat: 46.35475768005867, Lon: -70.26437825832838, RefDist: 122.51},
		},
	}
	layers := bsv.CalcLayers(a)
	fmt.Println("quebec cities layers:\n", layers)
}

func TestScanArea(t *testing.T) {
	lim := 56000
	br := &bsv.BoostRequest{
		MaxAge:    67,
		MinAge:    19,
		Genders:   []string{"male", "female"},
		Interests: []string{"kush", "money", "creatine", "coffee"},
		Limit:     lim,
	}

	a := &bsv.Area{
		Center: bsv.LatLon{Lat: 46.36683529877258, Lon: -72.58105744075569},
		Perim: []bsv.LatLon{
			{Lat: 44.9523555993882, Lon: -72.40506027364277, RefDist: 122.51},
			{Lat: 46.198697681289964, Lon: -74.10291529755564, RefDist: 122.51},
			{Lat: 47.23240565782047, Lon: -72.5834542471733, RefDist: 122.51},
			{Lat: 46.35475768005867, Lon: -70.26437825832838, RefDist: 122.51},
		},
	}

	bjsn, _ := json.MarshalIndent(&br, "", "    ")
	fmt.Println("boost request:\n", string(bjsn))
	u, n := bsv.ScanArea(context.Background(),
		a, br.Genders, br.Interests, br.MinAge, br.MaxAge, lim)
	fmt.Printf("found %d candidates\n", lim-n)
	for p, usrs := range u {
		fmt.Println("for place:", p)
		usrsJsn, _ := json.MarshalIndent(&usrs, "", "     ")
		fmt.Println(string(usrsJsn))
	}
}
