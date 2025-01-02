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
    // Add a delay to all requests
    time.Sleep(250 * time.Millisecond)
	return fetchUrlWithRetry(url, cache, false)
}

func fetchUrlWithRetry(url string, cache *mc.Client, isRetry bool) []byte {
	val, _, _, cacheReadErr := cache.Get(url)
	if cacheReadErr != nil {
	} else {
		log.Printf("Cache Hit: %s", url)
		return []byte(val)
	}

	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Println("Failed to create request")
		log.Println(err)
		return []byte{}
	}

	req.Header.Add("User-Agent", "hnt-hmrc")

	res, getErr := client.Do(req)
	if getErr != nil {
		log.Println("Request failed")
		log.Println(getErr)
		return []byte{}
	}

	log.Printf("%d - %s\n", res.StatusCode, url)
	if res.StatusCode != 200 && isRetry == false {
		log.Println("Retrying in 2 sec")
		time.Sleep(2 * time.Second)
		return fetchUrlWithRetry(url, cache, true)
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
		log.Printf("Failed to cache %s\n", url)
		log.Println("Cache write error: ", cacheWriteErr)
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

	if value < MIN_YEAR || value > MAX_YEAR {
		return 0, fmt.Errorf("%d is not a supported tax year", value)
	}

	return value, nil
}

func cacheKey(address string, taxYear int) string {
	return fmt.Sprintf("v1-%s-%d", address, taxYear)
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

	log.Printf("Fetching data ... %s\n", cacheKey(address, taxYear))
	data := getDataByAddress(address, cache, start, end)

	jsonData, err := json.Marshal(data)

	if err != nil {
		log.Printf("Failed to serialize JSON for cache %s", cacheKey(address, taxYear))
	}

	_, cacheError := cache.Set(cacheKey(address, taxYear), string(jsonData), 0, RESULT_CACHE_TTL, 0)
	if cacheError != nil {
		log.Printf("Cache failure %s %s", cacheKey(address, taxYear), cacheError)
	}

	log.Printf("Caching data %s", cacheKey(address, taxYear))
}
