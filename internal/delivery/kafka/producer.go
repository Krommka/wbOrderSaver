package kafka

import (
	"errors"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"wb_l0/configs"
)

var errUnknownType = errors.New("unknown event type")

type Producer struct {
	producer     *kafka.Producer
	flushTimeout int
}

func NewProducer(cfg *configs.Config) (*Producer, error) {
	conf := &kafka.ConfigMap{
		"bootstrap.servers": cfg.KF.BootstrapServers,
	}
	p, err := kafka.NewProducer(conf)
	if err != nil {
		return nil, fmt.Errorf("error creating the producer - %w", err)
	}
	return &Producer{producer: p, flushTimeout: cfg.KF.FlushTimeout}, nil
}

func (p *Producer) Produce(message, topic, key string) error {
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: []byte(message),
		Key:   []byte(key),
	}
	kafkaChan := make(chan kafka.Event)
	err := p.producer.Produce(kafkaMsg, kafkaChan)
	if err != nil {
		return fmt.Errorf("error sending message to kafka: %w", err)
	}
	e := <-kafkaChan
	switch ev := e.(type) {
	case kafka.Error:
		return fmt.Errorf("error while sending message to kafka: %w", ev)
	case *kafka.Message:
		return nil
	default:
		return errUnknownType
	}
}

func (p *Producer) Close() {
	p.producer.Flush(p.flushTimeout)
	p.producer.Close()
}
