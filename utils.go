package main

import (
	"encoding/json"
	"fmt"
	"github.com/memcachier/mc"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
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

	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println("Failed to create request")
		log.Println(err)
		return []byte{}
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Println("Request failed")
		log.Println(getErr)
		return []byte{}
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Println("Failed to extract response")
		log.Println(readErr)
		return []byte{}
	}

	_, cacheWriteErr := cache.Set(url, string(body), 0, URL_CACHE_TTL, 0)
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

func parseTaxYear(taxYear string) (int, error) {
	value, err := strconv.Atoi(taxYear)

	if err != nil {
		return 0, err
	}

	if value < 2020 || value > 2023 {
		return 0, fmt.Errorf("%d is not a supported tax year", value)
	}

	return value, nil
}

func cacheKey(address string, taxYear int) string {
	return fmt.Sprintf("%s-%d", address, taxYear)
}

func getDataByAddress(address string, cache *mc.Client, startTime time.Time, endTime time.Time) []DataPoint {
	var data []DataPoint

	priceData := getMarketData(cache, startTime, endTime)
	earnings := rewardsByDay(address, cache, startTime, endTime)

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

func fetchData(address string, taxYear int, cache *mc.Client) {
	tz, _ := time.LoadLocation("Europe/London")
	start := time.Date(taxYear, 4, 6, 0, 0, 0, 0, tz)
	end := time.Date(taxYear+1, 4, 6, 0, 0, 0, 0, tz)

	log.Printf("Fetching data %s", cacheKey(address, taxYear))
	data := getDataByAddress(address, cache, start, end)

	jsonData, err := json.Marshal(data)

	if err != nil {
		log.Printf("Failed to serialize JSON for cache %s", cacheKey(address, taxYear))
	}

	log.Printf("Attempting to caching data %d", len(data))
	_, cacheError := cache.Set(cacheKey(address, taxYear), string(jsonData), 0, RESULT_CACHE_TTL, 0)
	if cacheError != nil {
		log.Printf("Cache failure %s %s", cacheKey(address, taxYear), cacheError)
	}

	log.Printf("Caching data %s", cacheKey(address, taxYear))
}
