package main

import (
	"fmt"
	"log"
	"strconv"
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
}

func main() {
	err := dotenv.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := resty.New().EnableTrace()

	client.BaseURL = "https://sceneplus.webapis.loyaltysite.ca"
	client.SetHeader("authorization", fmt.Sprintf("Bearer %s", dotenv.GetString("SCENE_API_TOKEN")))
	client.SetHeader("origin", "https://www.sceneplus.ca")

	minDate, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00-08:00")
	maxDate, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00-08:00")

	pointsByCategory := map[Category]int{}
	for i := 0; true; i++ {
		resp, err := client.R().SetHeader("content-type", "application/json").SetBody(PointsHistoryReq{
			Types:      []PointType{PointTypeEarn},
			Categories: []Category{CategoryAll},
			Cards:      []string{"ALL"},
			FromDate:   "1900-01-01T00:00:00-08:00",
			ToDate:     time.Now().Format(time.RFC3339),
			Sort:       "DESC",
			Page:       i + 1,
		}).SetResult(&PointsHistoryResp{}).Post("/api/customer/points/history")
		if err != nil || resp.StatusCode() != 200 {
			log.Fatal(err)
		}

		history := resp.Result().(*PointsHistoryResp)

		isLast := false
		for _, point := range history.Data.PointsTransactions {
			if point.PointType != PointTypeEarn {
				continue
			}
			transDate, err := time.Parse(time.RFC3339, point.TransactionDate)
			if err != nil {
				log.Print(err)
			}
			if transDate.Before(minDate) {
				log.Println("Last trans date:", point.Description, point.TransactionDate, transDate, isLast)
				isLast = true
			}
			if isLast || transDate.After(maxDate) {
				log.Println("Skipping page", i+1)
				break
			}
			if point.PointType == PointTypeEarn {
				points, err := strconv.Atoi(point.Points)
				if err != nil {
					log.Print(err)
				}
				pointsByCategory[point.Categories[0]] += points
			}
		}
		if isLast {
			break
		}
	}

	totalPoints := 0
	for category, points := range pointsByCategory {
		totalPoints += points
		log.Println(category, points, fmt.Sprintf("%d", points/100))
	}
	log.Println("Total earned points:", totalPoints, fmt.Sprintf("%d", totalPoints/100))
}
