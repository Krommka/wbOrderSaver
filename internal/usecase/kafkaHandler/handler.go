package kafkaHandler

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/sirupsen/logrus"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) HandleMessage(message []byte, topic kafka.TopicPartition, cn int) error {
	logrus.Infof("Consumer: %d, message from kafka: %s with offset %d on partition %d",
		cn,
		string(message),
		topic.Offset,
		topic.Partition)
	return nil
}
