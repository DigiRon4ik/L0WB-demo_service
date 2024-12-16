// Package fake_order contains functions for generating fake orders.
package fake_order

import (
	"demo_service/internal/models"
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// GenerateFakeOrder generates a fake order.
func GenerateFakeOrder() models.Order {
	// const fn = "GenerateFakeOrder"
	if err := gofakeit.Seed(time.Now().UnixNano()); err != nil {
		fmt.Println(err)
	}

	uid := gofakeit.UUID()
	trackNumber := "WBILMTESTTRACK" + gofakeit.LetterN(10)
	address := gofakeit.Address()

	items := []models.Item{}
	for i := 0; i < gofakeit.Number(1, 4); i++ {
		items = append(items, models.Item{
			ChrtID:      gofakeit.Number(1, 10000000),
			TrackNumber: trackNumber,
			Price:       gofakeit.Number(1, 999999),
			RID:         gofakeit.UUID(),
			Name:        gofakeit.ProductName(),
			Sale:        gofakeit.Number(0, 100),
			Size:        gofakeit.RandomString([]string{"0", "1", "2", "3", "4", "5", "S", "M", "L", "XL", "XXL", "XXXL"}),
			TotalPrice:  gofakeit.Number(1, 10000000),
			NMID:        gofakeit.Number(1, 10000000),
			Brand:       gofakeit.Company(),
			Status:      gofakeit.HTTPStatusCode(),
		})
	}

	order := models.Order{
		OrderUID:    uid,
		TrackNumber: trackNumber,
		Entry:       gofakeit.RandomString([]string{"WBIL", "MEEST", "DHL"}),
		Delivery: models.Delivery{
			Name:    gofakeit.Name(),
			Phone:   gofakeit.PhoneFormatted(),
			Zip:     gofakeit.Zip(),
			City:    address.City,
			Address: address.Street,
			Region:  address.State,
			Email:   gofakeit.Email(),
		},
		Payment: models.Payment{
			Transaction:  uid,
			RequestID:    gofakeit.UUID(),
			Currency:     gofakeit.CurrencyShort(),
			Provider:     gofakeit.RandomString([]string{"wbpay", "applepay", "googlepay", "yandexpay", "sberpay"}),
			Amount:       gofakeit.Number(1, 1000),
			PaymentDT:    int(time.Now().Unix()),
			Bank:         gofakeit.RandomString([]string{"alpha", "tinkoff", "sberbank", "yo.money", "vtb", "raiffeisenbank", ""}),
			DeliveryCost: gofakeit.Number(0, 5000),
			GoodsTotal:   gofakeit.Number(0, 10000),
			CustomFee:    gofakeit.Number(0, 1000),
		},
		Items:             items,
		Locale:            gofakeit.LanguageAbbreviation(),
		InternalSignature: gofakeit.LetterN(5),
		CustomerID:        gofakeit.UUID(),
		DeliveryService:   gofakeit.RandomString([]string{"meest", "dhl", "wbil"}),
		Shardkey:          fmt.Sprintf("%d", gofakeit.Number(1, 1000000)),
		SmID:              gofakeit.Number(1, 1000000),
		DateCreated:       time.Now(),
		OofShard:          fmt.Sprintf("%d", gofakeit.Number(1, 1000000)),
	}

	return order
}
