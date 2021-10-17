package main

import (
	"github.com/gin-gonic/gin"
	// "github.com/heroku/x/hmetrics/onload"
	// 	"github.com/garfield-yin/gin-error-handler"
	"github.com/memcachier/mc"
	// 	"io"
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

	router.GET("/data/:address", func(c *gin.Context) {
		address := c.Param("address")
		taxYear, taxYearParseError := parseTaxYear(c.Query("tax_year"))

		if taxYearParseError != nil {
			c.JSON(400, gin.H{
				"error": "Invalid year provided",
			})
			return
		}

		tz, _ := time.LoadLocation("Europe/London")
		start := time.Date(taxYear, 4, 6, 0, 0, 0, 0, tz)
		end := time.Date(taxYear+1, 4, 6, 0, 0, 0, 0, tz)

		c.JSON(http.StatusOK, gin.H{
			"data": getDataByAddress(address, cache, start, end),
		})
	})

	// Get the balance of a HNT wallet
	router.GET("/balance/:address", func(c *gin.Context) {
		address := c.Param("address")

		balance := fetchBalance()

		c.JSON(http.StatusOK, gin.H{
			"balance": 0,
		})
	})

	// Get the price of a token pair
	router.GET("/price/:pair", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"price": 0,
		})
	})

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl.html", nil)
	})

	router.Run(":" + port)
}
