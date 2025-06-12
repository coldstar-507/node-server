package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	// "io"
	"net/http"
	"time"
)

var (
	timeFmt  = "2022-11-12T00:43:55.000Z"
	minFmt   = "2006-01-02 15:04:05"
	ogateurl = "https://api.orangegateway.com/graphql"
	method   = "POST"
)

func makeq(from, to time.Time, periodicity string) map[string]any {
	return map[string]any{
		"query": `query ($instrument_id: String!, $limit: Int, $date_range: DateRangeInput, $periodicity: InstrumentHistoryPeriodicity) {
   instrument_price_bars (instrument_id: $instrument_id, limit: $limit, date_range: $date_range, periodicity: $periodicity) { instrument_id, ts, close }}`,
		"variables": map[string]any{
			"instrument_id": "BSVUSD",
			"limit":         300,
			"data_range": map[string]any{
				"time_from": from.UTC().Format("2022-11-12T00:43:55.000Z"),
				"time_to":   to.UTC().Format("2022-11-12T00:43:55.000Z"),
			},
			"periodicity": periodicity,
		},
	}
}

func makeq_(from, to time.Time, periodicity string) ([]byte, error) {
	m := map[string]any{
		"query": `query ($instrument_id: String!, $limit: Int, $date_range: DateRangeInput, $periodicity: InstrumentHistoryPeriodicity) {
   instrument_price_bars (instrument_id: $instrument_id, limit: $limit, date_range: $date_range, periodicity: $periodicity) { instrument_id, ts, close }}`,
		"variables": map[string]any{
			"instrument_id": "BSVUSD",
			"limit":         300,
			"data_range": map[string]any{
				"time_from": from.UTC().Format(timeFmt),
				"time_to":   to.UTC().Format(timeFmt),
			},
			"periodicity": periodicity,
		},
	}
	return json.Marshal(m)
}

func updateFifteenRates() error {
	var from time.Time = time.Now().Add(time.Hour * 24 * 2 * -1)
	q, err := makeq_(from, time.Now(), "minute15")
	if err != nil {
		fmt.Println("updateFifteenRates: makeq error:", err)
		return err
	}

	res, err := http.Post(ogateurl, "application/json", bytes.NewReader(q))
	if err != nil {
		fmt.Println("updateFifteenRates: error making request:", err)
		return err
	}

	var js map[string]any
	err = json.NewDecoder(res.Body).Decode(&js)

	elements, ok := js["data"].(map[string]any)
	if !ok {
		err = fmt.Errorf("js[data] was expected to be a map")
		fmt.Println("updateFifteenRates: decoding res error:", err)
		return err
	}

	dataPoints, ok := elements["instrument_price_bars"].([]any)
	if !ok {
		err = fmt.Errorf("elements[instrument_price_bars] was expected to be a slice")
		fmt.Println("updateFifteenRates: decoding res error:", err)
		return err
	}

	slices.Reverse(dataPoints)
	fmt.Println(len(dataPoints))
	for _, x := range dataPoints {
		x_, ok := x.(map[string]any)
		if !ok {
			err = fmt.Errorf("expect x to be a map")
			return err
		}

		rate, ok := x_["close"].(float64)
		if !ok {
			err = fmt.Errorf("expect x_[close] to be a float64")
			return err
		}
		rate_ := float32(rate)

		timestr, ok := x_["ts"].(string)
		if !ok {
			err = fmt.Errorf("expect x_[ts] to be a string")
			return err
		}

		tt, err := time.Parse(minFmt, timestr)
		if err != nil {
			return err
		}

		ts := tt.UnixMilli() / 100
		fmt.Printf("time: %d, rate: %.2f\n", ts, rate_)
	}

	return nil
}

func main() {
	fmt.Println(updateFifteenRates())
	// url := "https://api.orangegateway.com/graphql"
	// method := "POST"

	// now := time.Now()
	// m3 := now.Add(time.Hour * 24 * 30 * -1)
	// per := "day"
	// q := makeq(m3, now, per)

	// b, err := json.Marshal(q)
	// if err != nil {
	// 	panic(err)
	// }

	// client := &http.Client{}
	// req, err := http.NewRequest(method, url, bytes.NewReader(b))

	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// req.Header.Add("Content-Type", "application/json")

	// res, err := client.Do(req)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// defer res.Body.Close()

	// var m map[string]any

	// err = json.NewDecoder(res.Body).Decode(&m)
	// if err != nil {
	// 	panic(err)
	// }

	// i, err := json.MarshalIndent(m, "", "   ")
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(string(i))
}
