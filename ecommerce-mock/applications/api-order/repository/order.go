package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-order/model"
)

var ErrNotFound = errors.New("order not found")

type OrderRepository struct {
	db *pgxpool.Pool
}

func Connect(dsn string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}

func Migrate(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS orders (
			id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			customer_id  UUID        NOT NULL,
			status       VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			items        JSONB       NOT NULL DEFAULT '[]',
			total_amount NUMERIC(10,2) NOT NULL,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
		CREATE INDEX IF NOT EXISTS idx_orders_status      ON orders(status);
	`)
	return err
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error) {
	itemsJSON, err := json.Marshal(req.Items)
	if err != nil {
		return nil, err
	}

	var total float64
	for _, item := range req.Items {
		total += item.Price * float64(item.Quantity)
	}

	var o model.Order
	var itemsRaw []byte
	err = r.db.QueryRow(ctx, `
		INSERT INTO orders (customer_id, items, total_amount)
		VALUES ($1, $2, $3)
		RETURNING id, customer_id, status, items, total_amount, created_at, updated_at
	`, req.CustomerID, itemsJSON, total).
		Scan(&o.ID, &o.CustomerID, &o.Status, &itemsRaw, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(itemsRaw, &o.Items); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	var o model.Order
	var itemsRaw []byte
	err := r.db.QueryRow(ctx, `
		SELECT id, customer_id, status, items, total_amount, created_at, updated_at
		FROM orders WHERE id = $1
	`, id).Scan(&o.ID, &o.CustomerID, &o.Status, &itemsRaw, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(itemsRaw, &o.Items); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepository) ListByCustomer(ctx context.Context, customerID string) ([]model.Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, customer_id, status, items, total_amount, created_at, updated_at
		FROM orders WHERE customer_id = $1
		ORDER BY created_at DESC
	`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		var itemsRaw []byte
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.Status, &itemsRaw, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(itemsRaw, &o.Items); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []model.Order{}
	}
	return orders, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, status model.OrderStatus) (*model.Order, error) {
	var o model.Order
	var itemsRaw []byte
	err := r.db.QueryRow(ctx, `
		UPDATE orders SET status = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, customer_id, status, items, total_amount, created_at, updated_at
	`, status, id).Scan(&o.ID, &o.CustomerID, &o.Status, &itemsRaw, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(itemsRaw, &o.Items); err != nil {
		return nil, err
	}
	return &o, nil
}
