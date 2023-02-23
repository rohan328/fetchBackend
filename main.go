package main

import (
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gopkg.in/validator.v2"
)

type Item struct {
	ShortDescription string `json:"shortDescription" binding:"required" validate:"regexp=^\\S+$"`
	Price            string `json:"price" binding:"required" validate:"regexp=^\\d+\\.\\d{2}$"`
}

type Receipt struct {
	ID           string `json:"id"`
	Retailer     string `json:"retailer" binding:"required" validate:"regexp=^[\s\S]+$"`
	PurchaseDate string `json:"purchaseDate" binding:"required" validate:"regexp=^\\d{4}\\-(0[1-9]|1[012])\\-(0[1-9]|[12][0-9]|3[01])$"`
	PurchaseTime string `json:"purchaseTime" binding:"required" validate:"regexp=^([0-9]|0[0-9]|1[0-9]|2[0-3]):[0-5][0-9]$"`
	Total        string `json:"total" binding:"required" validate:"regexp=^\\d+\\.\\d{2}$"`
	Items        []Item `json:"items" binding:"required" validate:"min=1"`
	Points       string `json:"points"`
}

var receipts = []Receipt{}

func CountAlphanum(str string) int {
	isAlphaNum := regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString
	count := 0
	for _, char := range str {
		if isAlphaNum(string(char)) {
			count++
		}
	}
	return count
}

func calculatePoints(receipt Receipt) string {
	points := 0

	// one point for each alphanumeric char in retailer name
	points += CountAlphanum(receipt.Retailer)

	//parse total to float
	total, parseTotalErr := strconv.ParseFloat(receipt.Total, 64)
	if parseTotalErr == nil {
		//50 points if total is round number
		if math.Mod(total, 1) == 0 {
			points += 50
		}

		//25 points if total is multiple of 0.25
		if math.Mod(total, 0.25) == 0 {
			points += 25
		}
	}

	// 5 points for every 2 items
	itemPoints := (len(receipt.Items) / 2) * 5
	points += itemPoints

	//points for item description
	for _, item := range receipt.Items {
		if len(strings.TrimSpace(item.ShortDescription))%3 == 0 {
			price, _ := strconv.ParseFloat(item.Price, 64)
			points += int(math.Ceil(price * 0.2))
		}
	}

	//6 points for odd dates
	dateSplit := strings.Split(receipt.PurchaseDate, "-")
	date, _ := strconv.ParseInt(dateSplit[2], 10, 32)
	if date%2 != 0 {
		points += 6
	}

	//10 points for purchase between 2pm and 4pm
	timeSplit := strings.Split(receipt.PurchaseTime, ":")
	hr, _ := strconv.ParseInt(timeSplit[0], 10, 32)
	min, _ := strconv.ParseInt(timeSplit[1], 10, 32)
	if hr >= 14 && min > 0 && hr < 16 {
		points += 10
	}

	return strconv.Itoa(points)
}

func processReceipt(c *gin.Context) {

	//create new receipt and read from body
	var receipt Receipt
	err := c.BindJSON(&receipt)
	if err != nil {
		//error reading receipt from body
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := validator.Validate(receipt); err != nil {
		//validation error
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	//generate unique ID
	receipt.ID = uuid.New().String()

	//calculate points
	receipt.Points = calculatePoints(receipt)

	//add receipt to receipts array
	receipts = append(receipts, receipt)

	//return statusOK with ID
	c.JSON(http.StatusOK, gin.H{
		"id": receipt.ID,
	})
}

func getPoints(c *gin.Context) {

	id := c.Param("id")
	for _, receipt := range receipts {
		if receipt.ID == id {
			c.JSON(http.StatusOK, gin.H{
				"points": receipt.Points,
			})
			return
		}
	}
	c.JSON(http.StatusNotFound, "No receipt found for that id")
}

func main() {
	router := gin.Default()
	router.GET("/receipt/:id/points", getPoints)
	router.POST("/receipt/process", processReceipt)

	router.Run("localhost:8080")
}
