package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"time"
	"wb_l0/configs"
	"wb_l0/configs/loader/dotEnvLoader"
	k "wb_l0/internal/delivery/kafka"
	"wb_l0/internal/domain"
)

func main() {

	envLoader := dotEnvLoader.DotEnvLoader{}
	cfg := configs.MustLoad(envLoader)
	cfg.KF.BootstrapServers = "localhost:9091, localhost:9092,localhost:9093"

	p, err := k.NewProducer(cfg)
	if err != nil {
		logrus.Fatal(err)
	}
	numberOfKeys := cfg.KF.ProducerNumberOfKeys
	uuids := generateKeys(numberOfKeys)
	for i := 1; i < 150; i++ {
		order := domain.CreateTestOrder(i)
		orderString, err := json.Marshal(order)
		if err != nil {
			fmt.Printf("error marshalling order %v: %v\n", order, err)
			os.Exit(1)
		}
		key := uuids[i%len(uuids)]
		if err = p.Produce(string(orderString), cfg.KF.Topic, key); err != nil {
			fmt.Printf("Error producing order %v: %v\n", order, err)
		}
		fmt.Printf("Producing order %v to Kafka\n", order)
	}
}

func generateKeys(numberOfKeys int) []string {
	keys := make([]string, numberOfKeys)
	for i := 0; i < numberOfKeys; i++ {
		keys[i] = uuid.NewString()
	}
	return keys
}

func intToHex20(num int) string {
	// Преобразуем число в HEX строку
	hexStr := strconv.FormatInt(int64(num), 16)

	// Дополняем строку лидирующими нулями до 20 символов
	if len(hexStr) < 20 {
		// Создаем строку из нулей нужной длины
		zeros := make([]byte, 20-len(hexStr))
		for i := range zeros {
			zeros[i] = '0'
		}
		hexStr = string(zeros) + hexStr
	}

	return hexStr
}

func createTestOrder() domain.Order {
	now := time.Now().UTC()
	return domain.Order{
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
		Delivery: domain.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: domain.Payment{
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
		Items: []domain.Item{
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
}
