package main

import (
	"github.com/memcachier/mc"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sort"
	"time"
)

type DataPoint struct {
	Date     string  `json:"date"`
	Earnings float64 `json:"earnings"`
	Tokens   float64 `json:"tokens"`
	Price    float64 `json:"price"`
}

func fetchUrl(url string, cache *mc.Client) []byte {

	val, _, _, cacheReadErr := cache.Get(url)
	if cacheReadErr != nil {
		log.Printf("Cache Miss: %s", url)
	} else {
		return []byte(val)
	}

	client := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	_, cacheWriteErr := cache.Set(url, string(body), 0, 86400, 0)
	if cacheWriteErr != nil {
		log.Printf("Failed to cache %s", url)
	}

	return body
}

func dateAtStartOfDay(date time.Time) time.Time {
	year, month, day := time.Time(date).Date()

	key := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

	return key
}

func getDataByAddress(address string, cache *mc.Client) []DataPoint {
	var data []DataPoint

	priceData := getMarketData(cache)
	earnings := rewardsByDay(address, cache)

	for date, earnt := range earnings {
		coinPrice := priceData[date]

		roundedEarnings := earnt / math.Pow(10, 8)
		formattedDate := date.Format("2006-01-02")

		entry := DataPoint{
			formattedDate,
			(coinPrice * roundedEarnings),
			roundedEarnings,
			coinPrice,
		}

		data = append(data, entry)

		sort.SliceStable(data, func(i, j int) bool {
			return data[i].Date < data[j].Date
		})
	}

	return data
}

// func getTestData() []DataPoint {
//   address := "13eqa627iye9dhSkxqFdyM2mVWWWTmb71biqNQgVHg8rCk4M3Zt"
//
//   return getDataByAddress(address)
// }
