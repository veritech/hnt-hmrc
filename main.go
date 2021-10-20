package main

import (
	"github.com/gin-gonic/gin"
	// "github.com/heroku/x/hmetrics/onload"
	// 	"github.com/garfield-yin/gin-error-handler"
	"github.com/memcachier/mc"
	// 	"io"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

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

func fetchData(address string, taxYear int, cache *mc.Client) {
	tz, _ := time.LoadLocation("Europe/London")
	start := time.Date(taxYear, 4, 6, 0, 0, 0, 0, tz)
	end := time.Date(taxYear+1, 4, 6, 0, 0, 0, 0, tz)

	log.Printf("Fetching data %s", address)
	data := getDataByAddress(address, cache, start, end)

	jsonData, err := json.Marshal(data)

	if err != nil {
		log.Printf("Failed to serialize JSON for cache")
	}

	log.Printf("Attempting to caching data")
	_, cacheError := cache.Set(cacheKey(address, taxYear), string(jsonData), 0, 600, 0)
	if cacheError != nil {
		log.Printf("Cache failure %s", cacheError)
	}
}

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
		cachedData, _, _, err := cache.Get(dataKey)

		if err != nil {
			log.Printf("Cache data found")
			var data []DataPoint
			json.Unmarshal([]byte(cachedData), &data)

			c.JSON(http.StatusOK, gin.H{
				"data": data,
			})
			cache.Del(dataKey)
		} else {
			log.Printf("Cache error %s", err)
			c.JSON(425, gin.H{
				"data": nil,
			})
		}
	})

	// Get the balance of a HNT wallet
	router.GET("/balance/:address", func(c *gin.Context) {
		address := c.Param("address")

		balance := fetchBalance(address, cache)

		c.JSON(http.StatusOK, gin.H{
			"balance": balance,
		})
	})

	// Get the price of a token pair
	router.GET("/price/:token", func(c *gin.Context) {
		token := c.Param("token")

		price, err := getMarketPrice(token, cache)

		if err != nil {
			c.JSON(400, gin.H{
				"error": "Bad token provided",
			})
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
