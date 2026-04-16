package model

import "time"

type Product struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Price        float64   `json:"price"`
	Category     string    `json:"category"`
	Stock        int       `json:"stock"`
	Discontinued bool      `json:"discontinued"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateProductRequest struct {
	Name         string  `json:"name" binding:"required"`
	Description  string  `json:"description"`
	Price        float64 `json:"price" binding:"required,gt=0"`
	Category     string  `json:"category"`
	Stock        int     `json:"stock" binding:"gte=0"`
	Discontinued bool    `json:"discontinued"`
}

type UpdateProductRequest struct {
	Name         *string  `json:"name"`
	Description  *string  `json:"description"`
	Price        *float64 `json:"price"`
	Category     *string  `json:"category"`
	Stock        *int     `json:"stock"`
	Discontinued *bool    `json:"discontinued"`
}

type ListResponse struct {
	Data  []Product `json:"data"`
	Total int       `json:"total"`
	Page  int       `json:"page"`
	Limit int       `json:"limit"`
}
