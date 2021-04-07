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

func (n *RewardTime) UnmarshalJSON(buf []byte) error {
	value := strings.Trim(string(buf), "\"")
	
	fmt.Println("RewardTime " + value)
	parsedDate, err := time.Parse(time.RFC3339Nano, value)
	fmt.Println("Parsed " + parsedDate)
	fmt.Println("============")
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

func fetchRewards(address string, cursor string, cache *mc.Client) ([]Reward, string) {
	format := "2006-01-01"
	tz, _ := time.LoadLocation("Europe/London")
	start := time.Date(2020, 4, 6, 0, 0, 0, 0, tz).Format(format)
	end := time.Date(2021, 4, 5, 23, 59, 59, 999, tz).Format(format)

	url := fmt.Sprintf("https://api.helium.io/v1/hotspots/%s/rewards?max_time=%s&min_time=%s&cursor=%s", address, end, start, cursor)

	response := fetchUrl(url, cache)

	rewardResponse := HotspotRewardsRewards{}

	json.Unmarshal(response, &rewardResponse)

	return rewardResponse.Data, rewardResponse.Cursor
}

func fetchAllRewards(address string, cache *mc.Client) []Reward {
	var allRewards []Reward

	var stop bool = false
	var nextCursor string = ""

	fmt.Println("fetching rewards")
	for !stop {
		rewards, cursor := fetchRewards(address, nextCursor, cache)

		stop = cursor == ""
		nextCursor = cursor

		allRewards = append(allRewards, rewards...)
	}
	fmt.Println("fetched rewards")

	return allRewards
}

func fetchAllRewardsForAllHotspots(address string, cache *mc.Client) []Reward {
	hotspots := fetchHotspots(address, cache)
	var allRewards []Reward

	for _, item := range hotspots {
		rewards := fetchAllRewards(item.Address, cache)

		allRewards = append(allRewards, rewards...)
	}

	return allRewards
}

func rewardsByDay(address string, cache *mc.Client) EarningsByDay {
	allRewards := fetchAllRewardsForAllHotspots(address, cache)

	earnings := make(EarningsByDay)
	
	for _, reward := range allRewards {
		key := dateAtStartOfDay(time.Time(reward.Timestamp))

		if val, ok := earnings[key]; ok {
			earnings[key] = val + reward.Amount
		} else {
			earnings[key] = reward.Amount
		}
	}
	
	fmt.Printf("Rewards %+v", allRewards)
	fmt.Println("")
	fmt.Printf("Earnings %+v", earnings)
	fmt.Println("")

	return earnings
}
