// Sample sqlc-generated basic types
package db

import (
	"database/sql"
	"time"
)

// User represents a user in the system
type User struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	Email     string      `json:"email"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty"`
	IsActive  bool        `json:"is_active"`
}

// Product represents a product in the catalog
type Product struct {
	ID          int64         `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Price       float64       `json:"price"`
	InStock     bool          `json:"in_stock"`
	SKU         sql.NullString `json:"sku,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
}
