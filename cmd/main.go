package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/profclems/go-dotenv"
)

type PointsHistoryResp struct {
	Data             Data        `json:"data"`
	ValidationErrors interface{} `json:"validationErrors"`
}

type Data struct {
	PointsTransactions []PointsTransaction `json:"pointsHistory"`
	ItemsCount         int64               `json:"itemsCount"`
	TotalItemCount     int64               `json:"totalItemCount"`
	PageNumber         int64               `json:"pageNumber"`
	ErrorDetails       interface{}         `json:"errorDetails"`
}

type PointsTransaction struct {
	PointID           string       `json:"pointId"`
	PointType         PointType    `json:"pointCategory"`
	Description       string       `json:"description"`
	Location          *string      `json:"location"`
	Brand             interface{}  `json:"brand"`
	Points            string       `json:"points"`
	Categories        []Category   `json:"categories"`
	Multiplier        *string      `json:"multiplier"`
	TransactionAmount string       `json:"transactionAmount"`
	PointDate         string       `json:"pointDate"`
	TransactionDate   string       `json:"transactionDate"`
	Card              Card         `json:"card"`
	AwardType         interface{}  `json:"awardType"`
	PartnerCode       PartnerCode  `json:"partnerCode"`
	IconTypeCode      IconTypeCode `json:"iconTypeCode"`
}

type Card string

const (
	Axg   Card = "AXG"
	Scene Card = "SCENE"
)

type Category string

const (
	CategoryAll           Category = "ALL"
	CategoryDining        Category = "DINING"
	CategoryMovies        Category = "MOVIES"
	CategoryShopping      Category = "SHOPPING"
	CategoryEntertainment Category = "ENTERTAINMENT"
	CategoryTransit       Category = "TRANSIT"
	CategoryGroceries     Category = "GROCERIES"
	CategoryTravel        Category = "TRAVEL"
	CategoryStreaming     Category = "STREAMING"
	CategoryGas           Category = "GAS"
	CategoryOther         Category = "OTHER"
)

type IconTypeCode string

const (
	BnsTransaction     IconTypeCode = "BNS_TRANSACTION"
	PartnerTransaction IconTypeCode = "PARTNER_TRANSACTION"
)

type PartnerCode string

const (
	Bns    PartnerCode = "BNS"
	Sobeys PartnerCode = "SOBEYS"
)

type PointType string

const (
	PointTypeAll        PointType = "ALL"
	PointTypeEarn       PointType = "EARN"
	PointTypeRedeem     PointType = "REDEEM"
	PointTypeAdjustment PointType = "ADJUSTMENT"
	PointTypeTransfer   PointType = "TRANSFER"
	PointTypeReverse    PointType = "REVERSE"
)

type PointsHistoryReq struct {
	Types      []PointType `json:"Types"`
	Categories []Category  `json:"Categories"`
	Cards      []string    `json:"Cards"`
	FromDate   string      `json:"FromDate"`
	ToDate     string      `json:"ToDate"`
	Page       int         `json:"Page"`
	Sort       string      `json:"Sort"`
	Limit      int         `json:"Limit"`
}

const PAGE_SIZE = 100

func main() {
	err := dotenv.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := resty.New().EnableTrace()

	client.BaseURL = "https://sceneplus.webapis.loyaltysite.ca"
	client.SetHeader("authorization", fmt.Sprintf("Bearer %s", dotenv.GetString("SCENE_API_TOKEN")))
	client.SetHeader("origin", "https://www.sceneplus.ca")

	startDateStr := flag.String("start-date", "", "Start of transaction date range")
	endDateStr := flag.String("end-date", time.Now().Format(time.DateOnly), "End of transaction date range")
	flag.Parse()

	minDate, err := time.Parse(time.DateOnly, *startDateStr)
	if err != nil {
		log.Fatal("Invalid start date")
	}
	maxDate, err := time.Parse(time.DateOnly, *endDateStr)
	if err != nil {
		log.Fatal("Invalid end date")
	}

	pointsByCategory := map[Category]int{}
	totalPoints := 0

	totalPages := 1
	for i := 0; i < totalPages; i++ {
		resp, err := client.R().SetHeader("content-type", "application/json").SetBody(PointsHistoryReq{
			Types:      []PointType{PointTypeEarn},
			Categories: []Category{CategoryAll},
			Cards:      []string{"ALL"},
			FromDate:   minDate.Format(time.DateOnly),
			ToDate:     maxDate.Format(time.DateOnly),
			Sort:       "ASC",
			Page:       i + 1,
			Limit:      PAGE_SIZE,
		}).SetResult(&PointsHistoryResp{}).Post("/api/customer/points/history")
		if err != nil {
			log.Fatal(err)
		}
		if resp.StatusCode() != 200 {
			log.Fatal(strings.Join([]string{"failed request", fmt.Sprint(resp.StatusCode()), string(resp.Body())}, "\n"))
		}

		history := resp.Result().(*PointsHistoryResp)

		totalPages = int(math.Ceil(float64(history.Data.TotalItemCount / PAGE_SIZE)))

		for _, point := range history.Data.PointsTransactions {
			if !strings.EqualFold(string(point.PointType), string(PointTypeEarn)) {
				continue
			}
			points, err := strconv.Atoi(point.Points)
			if err != nil {
				log.Print(err)
			}
			pointsByCategory[point.Categories[0]] += points
			totalPoints += points
		}
	}

	for category, points := range pointsByCategory {
		log.Println(category, fmt.Sprintf("%.2f%%", float64(points)/float64(totalPoints)*100), points, fmt.Sprintf("$%.2f", float64(points)/100))
	}
	log.Println("TOTAL", totalPoints, fmt.Sprintf("$%.2f", float64(totalPoints)/100))
}
