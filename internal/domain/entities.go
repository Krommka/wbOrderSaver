package domain

import (
	"time"
)

type Order struct {
	OrderUID          string    `json:"order_uid" validate:"required,order_uid"`
	TrackNumber       string    `json:"track_number" validate:"required,track_number"`
	Entry             string    `json:"entry" validate:"required,alpha,min=3,max=10"`
	Delivery          Delivery  `json:"delivery" validate:"required"`
	Payment           Payment   `json:"payment" validate:"required"`
	Items             []Item    `json:"items" validate:"required,min=1,dive"`
	Locale            string    `json:"locale" validate:"required,oneof=en ru"`
	InternalSignature string    `json:"internal_signature" validate:"max=255"`
	CustomerID        string    `json:"customer_id" validate:"required,alphanum,max=50"`
	DeliveryService   string    `json:"delivery_service" validate:"required,alpha,max=50"`
	ShardKey          string    `json:"shardkey" validate:"required,alphanum,max=10"`
	SMID              int       `json:"sm_id" validate:"required,min=0"`
	DateCreated       time.Time `json:"date_created" validate:"required"`
	OOFShard          string    `json:"oof_shard" validate:"required,numeric,max=10"`
}

type Delivery struct {
	Name    string `json:"name" validate:"required,min=2,max=100"`
	Phone   string `json:"phone" validate:"required,phone"`
	Zip     string `json:"zip" validate:"required,zip,max=20"`
	City    string `json:"city" validate:"required,min=2,max=100"`
	Address string `json:"address" validate:"required,min=5,max=255"`
	Region  string `json:"region" validate:"required,min=2,max=100"`
	Email   string `json:"email" validate:"required,email,max=100"`
}

type Payment struct {
	Transaction  string `json:"transaction" validate:"required,order_uid"`
	RequestID    string `json:"request_id" validate:"max=50"`
	Currency     string `json:"currency" validate:"required,currency"`
	Provider     string `json:"provider" validate:"required,alpha,max=50"`
	Amount       int    `json:"amount" validate:"required,min=0"`
	PaymentDT    int64  `json:"payment_dt" validate:"required,min=0"`
	Bank         string `json:"bank" validate:"required,alpha,max=100"`
	DeliveryCost int    `json:"delivery_cost" validate:"min=0"`
	GoodsTotal   int    `json:"goods_total" validate:"min=0"`
	CustomFee    int    `json:"custom_fee" validate:"min=0"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id" validate:"required,min=1"`
	TrackNumber string `json:"track_number" validate:"required,track_number"`
	Price       int    `json:"price" validate:"required,min=1"`
	RID         string `json:"rid" validate:"required,min=10,max=50"`
	Name        string `json:"name" validate:"required,min=2,max=255"`
	Sale        int    `json:"sale" validate:"min=0,max=100"`
	Size        string `json:"size" validate:"required,max=50"`
	TotalPrice  int    `json:"total_price" validate:"min=0"`
	NMID        int    `json:"nm_id" validate:"required,min=1"`
	Brand       string `json:"brand" validate:"required,min=2,max=255"`
	Status      int    `json:"status" validate:"required,min=100,max=600"`
}
