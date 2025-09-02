package kafkaHandler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"log/slog"
	"time"
	"wb_l0/internal/domain"
	"wb_l0/internal/usecase"
)

type KafkaHandler struct {
	orderUsecase *usecase.OrderUsecase
	log          *slog.Logger
}

func NewKafkaHandler(orderUsecase *usecase.OrderUsecase, log *slog.Logger) *KafkaHandler {
	return &KafkaHandler{
		orderUsecase,
		log,
	}
}

func (h *KafkaHandler) HandleMessage(message []byte, topic kafka.TopicPartition, cn int) error {
	startTime := time.Now()

	h.log.Debug("Kafka message received",
		"topic", topic.Topic,
		"partition", topic.Partition,
		"offset", topic.Offset,
		"consumer", cn,
		"message_size", len(message),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	order, err := h.parseOrder(message)
	if err != nil {
		h.log.Error("Failed to parse order",
			"error", err,
			"topic", topic.Topic,
			"partition", topic.Partition,
			"offset", topic.Offset,
			"consumer", cn,
			"message_size", len(message),
		)
		return nil
	}

	if err = h.orderUsecase.CreateOrder(ctx, order); err != nil {
		h.log.Error("Failed to create order",
			"order_uid", order.OrderUID,
			"error_type", "transport",
			"error", err,
			"topic", topic.Topic,
			"partition", topic.Partition,
			"offset", topic.Offset,
			"consumer", cn,
			"message_size", len(message),
		)
		return err
	}
	h.log.Info("Message processing completed",
		"status", "success",
		"order_uid", order.OrderUID,
		"processing_time_ms", time.Since(startTime).Milliseconds(),
	)

	return nil
}

func (h *KafkaHandler) parseOrder(message []byte) (domain.Order, error) {
	var order domain.Order

	if err := json.Unmarshal(message, &order); err != nil {
		return domain.Order{}, fmt.Errorf("json unmarshal failed: %w", err)
	}

	return order, nil
}
