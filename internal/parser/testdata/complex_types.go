// Sample sqlc-generated complex types
package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Order represents an order in the system
type Order struct {
	ID            int64          `json:"id"`
	CustomerID    int64          `json:"customer_id"`
	OrderDate     time.Time      `json:"order_date"`
	Status        string         `json:"status"`
	Total         float64        `json:"total"`
	Items         []OrderItem    `json:"items"`
	ShippingInfo  *ShippingInfo  `json:"shipping_info,omitempty"`
	Notes         sql.NullString `json:"notes,omitempty"`
	PaymentMethod sql.NullString `json:"payment_method,omitempty"`
}

// OrderItem represents an item within an order
type OrderItem struct {
	ID        int64   `json:"id"`
	OrderID   int64   `json:"order_id"`
	ProductID int64   `json:"product_id"`
	Quantity  int32   `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Subtotal  float64 `json:"subtotal"`
}

// ShippingInfo contains shipping details
type ShippingInfo struct {
	Address     string    `json:"address"`
	City        string    `json:"city"`
	PostalCode  string    `json:"postal_code"`
	Country     string    `json:"country"`
	TrackingNum string    `json:"tracking_num,omitempty"`
	ShippedAt   time.Time `json:"shipped_at,omitempty"`
}

// OrderStatus represents an enum type
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Document demonstrates UUID, JSONB, and custom types
type Document struct {
	ID        uuid.UUID       `json:"id"`
	Title     string          `json:"title"`
	Content   []byte          `json:"content"`              // For BYTEA/BLOB data
	Metadata  json.RawMessage `json:"metadata,omitempty"`   // For JSON/JSONB
	Tags      []string        `json:"tags"`                 // For array types
	CreatedBy uuid.NullUUID   `json:"created_by,omitempty"` // Nullable UUID
	Version   int32           `json:"version"`
}

// Transaction demonstrates decimal types and enums
type Transaction struct {
	ID            uuid.UUID       `json:"id"`
	Amount        decimal.Decimal `json:"amount"` // For DECIMAL/NUMERIC
	Currency      string          `json:"currency"`
	Status        OrderStatus     `json:"status"` // Enum type
	ReferenceCode sql.NullString  `json:"reference_code,omitempty"`
	ProcessedAt   time.Time       `json:"processed_at"`
	Attachments   [][]byte          `json:"attachments,omitempty"` // For BYTEA/BLOB
}

// Configuration demonstrates complex JSON handling
type Configuration struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	Settings     json.RawMessage `json:"settings"` // For JSONB
	IsActive     bool            `json:"is_active"`
	ValidFrom    time.Time       `json:"valid_from"`
	ValidTo      sql.NullTime    `json:"valid_to,omitempty"`
	NumericArray []int32         `json:"numeric_array"` // For INT[] array type
	StringArray  []string        `json:"string_array"`  // For TEXT[] array type
}

// Necessary for pgx/v5 handling of UUID
type NullUUID struct {
	UUID  uuid.UUID
	Valid bool
}
