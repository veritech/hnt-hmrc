package main

import (
	"encoding/json"
	"fmt"
	"github.com/memcachier/mc"
	"log"
	"strings"
	"time"
)

type RewardTime time.Time

type EarningsByDay map[time.Time]float64

// helium api
type Reward struct {
	Account   string     `json:"account"`
	Amount    float64    `json:"amount"`
	Timestamp RewardTime `json:"timestamp"`
}

type Hotspot struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type AccountHotspotsResponse struct {
	Data []Hotspot `json:"data"`
}

type HotspotRewardsRewards struct {
	Data   []Reward `json:"data"`
	Cursor string   `json:"cursor"`
}

type AddressData struct {
	Balance int `json:balance`
}

type AddressResponse struct {
	Data AddressData `json:data`
}

func (n *RewardTime) UnmarshalJSON(buf []byte) error {
	value := strings.Trim(string(buf), "\"")

	parsedDate, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		log.Println("Date parser", err)
		return err
	}

	*n = RewardTime(parsedDate)

	return nil
}

func fetchHotspots(address string, cache *mc.Client) []Hotspot {
	url := fmt.Sprintf("https://api.helium.io/v1/accounts/%s/hotspots", address)

	response := fetchUrl(url, cache)

	hotspots := AccountHotspotsResponse{}

	json.Unmarshal(response, &hotspots)

	return hotspots.Data
}

func fetchRewards(address string, cursor string, cache *mc.Client, startTime time.Time, endTime time.Time) ([]Reward, string) {
	format := "2006-01-02"

	url := fmt.Sprintf(
		"https://api.helium.io/v1/hotspots/%s/rewards?max_time=%s&min_time=%s&cursor=%s",
		address,
		endTime.Format(format),
		startTime.Format(format),
		cursor)

	response := fetchUrl(url, cache)

	rewardResponse := HotspotRewardsRewards{}

	json.Unmarshal(response, &rewardResponse)

	return rewardResponse.Data, rewardResponse.Cursor
}

func fetchAllRewards(address string, cache *mc.Client, startTime time.Time, endTime time.Time) []Reward {
	var allRewards []Reward

	var stop bool = false
	var nextCursor string = ""

	fmt.Println("fetching rewards")
	for !stop {
		rewards, cursor := fetchRewards(address, nextCursor, cache, startTime, endTime)

		stop = cursor == ""
		nextCursor = cursor

		allRewards = append(allRewards, rewards...)
	}
	fmt.Println("fetched rewards")

	return allRewards
}

func fetchAllRewardsForAllHotspots(address string, cache *mc.Client, startTime time.Time, endTime time.Time) []Reward {
	hotspots := fetchHotspots(address, cache)
	var allRewards []Reward

	for _, item := range hotspots {
		rewards := fetchAllRewards(item.Address, cache, startTime, endTime)

		allRewards = append(allRewards, rewards...)
	}

	return allRewards
}

func rewardsByDay(address string, cache *mc.Client, startTime time.Time, endTime time.Time) EarningsByDay {
	allRewards := fetchAllRewardsForAllHotspots(address, cache, startTime, endTime)

	earnings := make(EarningsByDay)

	for _, reward := range allRewards {
		key := dateAtStartOfDay(time.Time(reward.Timestamp))

		if val, ok := earnings[key]; ok {
			earnings[key] = val + reward.Amount
		} else {
			earnings[key] = reward.Amount
		}
	}

	return earnings
}

func fetchBalance(address string, cache *mc.Client) float64 {
	url := fmt.Sprintf("https://api.helium.io/v1/accounts/%s", address)

	response := fetchUrl(url, cache)

	responseObject := AddressResponse{}

	json.Unmarshal(response, &responseObject)

	if responseObject.Data.Balance == 0 {
		return 0
	}

	return float64(responseObject.Data.Balance) / 100000000
}
