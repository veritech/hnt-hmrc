package main

import (
	"encoding/json"
	"strconv"
	"time"
)

type PricesBytime = map[time.Time]float64

type PriceTime time.Time

type PriceTimeTuple struct {
	Price     float64
	Timestamp PriceTime
}

type MarketChart struct {
	Prices       []PriceTimeTuple `json:"prices"`
	MarketCaps   []PriceTimeTuple `json:"market_caps"`
	TotalVolumes []PriceTimeTuple `json:"total_volumes"`
}

func (n *PriceTime) UnmarshalJSON(buf []byte) error {
	value, _ := strconv.ParseInt(string(buf), 10, 64)

	date := time.Unix(value/1000, 0)

	*n = PriceTime(date)

	return nil
}

func (n *PriceTimeTuple) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&n.Timestamp, &n.Price}

	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}

	return nil
}

func convert(tuples []PriceTimeTuple) PricesBytime {
	hash := make(PricesBytime)

	for _, item := range tuples {
		key := dateAtStartOfDay(time.Time(item.Timestamp))

		hash[key] = item.Price
	}

	return hash
}

func getMarketData() PricesBytime {
	url := "https://api.coingecko.com/api/v3/coins/helium/market_chart?vs_currency=gbp&days=180&interval=daily"

	response := fetchUrl(url)

	var marketData MarketChart

	json.Unmarshal(response, &marketData)

	hash := convert(marketData.Prices)

	return hash
}
