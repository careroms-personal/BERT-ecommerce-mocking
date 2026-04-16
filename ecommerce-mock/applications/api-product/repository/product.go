package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-product/model"
)

var ErrNotFound = errors.New("product not found")

type ProductRepository struct {
	db *pgxpool.Pool
}

func Connect(dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.LazyConnect = true
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func Migrate(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS products (
			id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name         VARCHAR(255) NOT NULL,
			description  TEXT NOT NULL DEFAULT '',
			price        NUMERIC(10,2) NOT NULL,
			category     VARCHAR(100) NOT NULL DEFAULT '',
			stock        INTEGER NOT NULL DEFAULT 0,
			discontinued BOOLEAN NOT NULL DEFAULT FALSE,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
		CREATE INDEX IF NOT EXISTS idx_products_stock    ON products(stock);
	`)
	return err
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) List(ctx context.Context, page, limit int, category string) ([]model.Product, int, error) {
	offset := (page - 1) * limit

	countQuery := `SELECT COUNT(*) FROM products`
	listQuery := `
		SELECT id, name, description, price, category, stock, discontinued, created_at, updated_at
		FROM products
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	args := []any{limit, offset}

	if category != "" {
		countQuery = `SELECT COUNT(*) FROM products WHERE category = $1`
		listQuery = `
			SELECT id, name, description, price, category, stock, discontinued, created_at, updated_at
			FROM products
			WHERE category = $3
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = append(args, category)
	}

	var total int
	if category != "" {
		r.db.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE category = $1`, category).Scan(&total)
	} else {
		r.db.QueryRow(ctx, countQuery).Scan(&total)
	}

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.Stock, &p.Discontinued, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	if products == nil {
		products = []model.Product{}
	}
	return products, total, nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (*model.Product, error) {
	var p model.Product
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, price, category, stock, discontinued, created_at, updated_at
		FROM products WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.Stock, &p.Discontinued, &p.CreatedAt, &p.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) Create(ctx context.Context, req model.CreateProductRequest) (*model.Product, error) {
	var p model.Product
	err := r.db.QueryRow(ctx, `
		INSERT INTO products (name, description, price, category, stock, discontinued)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, description, price, category, stock, discontinued, created_at, updated_at
	`, req.Name, req.Description, req.Price, req.Category, req.Stock, req.Discontinued).
		Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.Stock, &p.Discontinued, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) Update(ctx context.Context, id string, req model.UpdateProductRequest) (*model.Product, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Price != nil {
		existing.Price = *req.Price
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}
	if req.Stock != nil {
		existing.Stock = *req.Stock
	}
	if req.Discontinued != nil {
		existing.Discontinued = *req.Discontinued
	}

	var p model.Product
	err = r.db.QueryRow(ctx, `
		UPDATE products
		SET name=$1, description=$2, price=$3, category=$4, stock=$5, discontinued=$6, updated_at=NOW()
		WHERE id=$7
		RETURNING id, name, description, price, category, stock, discontinued, created_at, updated_at
	`, existing.Name, existing.Description, existing.Price, existing.Category, existing.Stock, existing.Discontinued, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Category, &p.Stock, &p.Discontinued, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProductRepository) GetStock(ctx context.Context, id string) (int, error) {
	var stock int
	err := r.db.QueryRow(ctx, `SELECT stock FROM products WHERE id = $1`, id).Scan(&stock)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	return stock, nil
}

func (r *ProductRepository) DeductStock(ctx context.Context, id string, qty int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE products SET stock = stock - $1, updated_at = NOW()
		WHERE id = $2 AND stock >= $1
	`, qty, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("insufficient stock")
	}
	return nil
}
