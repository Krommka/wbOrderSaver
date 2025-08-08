package main

import (
	k "KafkaExample/internal/kafka"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	topic        = "my-topic"
	numberOfKeys = 20
)

var address = []string{"localhost:9091", "localhost:9092", "localhost:9093"}

func main() {
	p, err := k.NewProducer(address)
	if err != nil {
		logrus.Fatal(err)
	}
	uuids := generateUUID()
	for i := 0; i < 1000; i++ {
		msg := fmt.Sprintf("kafka message: %d", i)
		key := uuids[i%numberOfKeys]
		if err = p.Produce(msg, topic, key); err != nil {
			logrus.Error(err)
		}
	}
}

func generateUUID() [numberOfKeys]string {
	var uuids [numberOfKeys]string
	for i := 0; i < numberOfKeys; i++ {
		uuids[i] = uuid.NewString()
	}
	return uuids
}
