package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/memcachier/mc"
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

type Coin struct {
	Identifier string `json:"id"`
	Symbol     string `json:"symbol"`
	Name       string `json:"name"`
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

func getMarketData(cache *mc.Client, startTime time.Time, endTime time.Time) PricesBytime {
	url := fmt.Sprintf(
		"https://api.coingecko.com/api/v3/coins/helium/market_chart/range?vs_currency=GBP&from=%d&to=%d",
		startTime.Unix(),
		endTime.Unix())

	response := fetchUrl(url, cache)

	var marketData MarketChart

	json.Unmarshal(response, &marketData)

	hash := convert(marketData.Prices)

	return hash
}

func getIdentifierBySymbolMap(cache *mc.Client) (map[string]string, map[string]bool) {
	response := fetchUrl("https://api.coingecko.com/api/v3/coins/list", cache)

	var coins []Coin

	json.Unmarshal(response, &coins)

	identifierBySymbol := make(map[string]string)

	boolByIdentifier := make(map[string]bool)

	for _, coin := range coins {
		boolByIdentifier[coin.Identifier] = true

		identifierBySymbol[coin.Symbol] = coin.Identifier
	}

	return identifierBySymbol, boolByIdentifier
}

func getValidIdentifier(symbolOrIdentifier string, cache *mc.Client) string {
	input := strings.ToLower(symbolOrIdentifier)

	identifierBySymbol, boolByIdentifier := getIdentifierBySymbolMap(cache)

	if _, ok := boolByIdentifier[input]; ok {
		return input
	}

	return identifierBySymbol[input]
}

func getMarketPrice(tickerOrIdentifier string, cache *mc.Client) (float64, error) {
	identifier := getValidIdentifier(tickerOrIdentifier, cache)
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=gbp", identifier)

	response := fetchUrl(url, cache)

	var rootObject map[string]json.RawMessage
	rootObjectError := json.Unmarshal(response, &rootObject)

	if rootObjectError != nil {
		log.Println(rootObjectError)
		return 0, rootObjectError
	}

	var currencyValue map[string]float64
	currencyValueError := json.Unmarshal(rootObject[identifier], &currencyValue)
	if currencyValueError != nil {
		log.Printf("Parsing failed: %s %s", currencyValueError, string(response))
		return 0, currencyValueError
	}

	return currencyValue["gbp"], nil
}
