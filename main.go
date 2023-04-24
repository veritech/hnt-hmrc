package main

import (
	"github.com/gin-gonic/gin"
	// "github.com/heroku/x/hmetrics/onload"
	// 	"github.com/garfield-yin/gin-error-handler"
	"github.com/memcachier/mc"
	// 	"io"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

const RESULT_CACHE_TTL = 86400
const URL_CACHE_TTL = 3600
const MAX_YEAR = 2024
const MIN_YEAR = 2020

func main() {
	username := os.Getenv("MEMCACHIER_USERNAME")
	password := os.Getenv("MEMCACHIER_PASSWORD")
	server := os.Getenv("MEMCACHIER_SERVERS")

	cache := mc.NewMC(server, username, password)
	defer cache.Quit()

	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	router := gin.New()
	router.Use(gin.Logger())
	// 	router.Use(ginerror.ErrorHandle(errWriter))
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	// enqueue a job
	router.GET("/enqueue/:address", func(c *gin.Context) {
		address := c.Param("address")
		taxYear, taxYearParseError := parseTaxYear(c.Query("tax_year"))

		if taxYearParseError != nil {
			c.JSON(400, gin.H{
				"error": "Invalid year provided",
			})
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"enqueued": true,
		})

		// return early
		c.Abort()

		// Do we have cached data for this request?
		dataKey := cacheKey(address, taxYear)
		_, _, _, cacheReadErr := cache.Get(dataKey)

		if cacheReadErr == nil {
			log.Println("Cached hit, skipping processing")
			return
		}

		// Fetch the data async
		go fetchData(address, taxYear, cache)
	})

	// get the data
	router.GET("/data/:address", func(c *gin.Context) {
		address := c.Param("address")
		taxYear, taxYearParseError := parseTaxYear(c.Query("tax_year"))

		if taxYearParseError != nil {
			c.JSON(400, gin.H{
				"error": "Invalid year provided",
			})
			c.Abort()
			return
		}

		dataKey := cacheKey(address, taxYear)
		cachedData, _, _, cacheReadErr := cache.Get(dataKey)

		if cacheReadErr != nil {
			log.Printf("Cache error %s", cacheReadErr)
			c.JSON(425, gin.H{
				"data": nil,
			})
			c.Abort()
			return
		}

		log.Printf("Cache data found")
		var data []DataPoint
		json.Unmarshal([]byte(cachedData), &data)

		c.JSON(http.StatusOK, gin.H{
			"data": data,
		})
	})

	// Get the balance of a HNT wallet
	router.GET("/balance/:address", func(c *gin.Context) {
		address := c.Param("address")

		balance := fetchBalance(address, cache)

		c.JSON(http.StatusOK, gin.H{
			"balance": balance,
		})
	})

	router.GET("/solana/balance/:address/:token", func(c *gin.Context) {
		address := c.Param("address")
		token := c.Param("token")

		balance, err := fetchSolanaAccountBalance(address, token, cache)

		if err != nil {
			log.Printf("Unable to fetch solana balance %s %s %s", address, token, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": err,
			})
			c.Abort()
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"balance": balance,
			})
		}
	})

	// Get the price of a token pair
	router.GET("/price/:token", func(c *gin.Context) {
		token := c.Param("token")

		price, err := getMarketPrice(token, cache)

		if err != nil {
			log.Printf("Unable to get the price for %s %v", token, err)
			c.JSON(400, gin.H{
				"error": "Bad token provided",
			})
			c.Abort()
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"price": price,
		})
	})

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	router.Run(":" + port)
}
