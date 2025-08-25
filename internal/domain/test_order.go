package domain

import (
	"strconv"
	"time"
)

func CreateTestOrder(id int) Order {
	now := time.Now().UTC()
	order := Order{
		OrderUID:          "",
		TrackNumber:       "WBILMTESTTRACK",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SMID:              99,
		DateCreated:       now,
		OOFShard:          "1",
		Delivery: Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: Payment{
			Transaction:  "",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDT:    now.Unix(),
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []Item{
			{
				ChrtID:      9934930,
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				RID:         "ab4219087a764ae0b473",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NMID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
	}
	transaction := intToHex20(id)
	order.OrderUID = transaction
	order.Payment.Transaction = transaction

	return order
}

func intToHex20(num int) string {
	hexStr := strconv.FormatInt(int64(num), 16)

	if len(hexStr) < 20 {
		zeros := make([]byte, 20-len(hexStr))
		for i := range zeros {
			zeros[i] = '0'
		}
		hexStr = string(zeros) + hexStr
	}

	return hexStr
}
