// Seeds: api-cart (cart_db)
// Run: mongosh "mongodb://ecommerce:ecommerce_pass@localhost:27017/cart_db?authSource=cart_db" seeds/api-cart.js
//
// Customer UUIDs must match seeds/api-customer.sql:
//   customer_1 → 22222222-2222-2222-2222-000000000001
//   customer_2 → 22222222-2222-2222-2222-000000000002
//   fake       → 99999999-9999-9999-9999-999999999999  (no matching customer — for error simulation)
//
// Product UUIDs must match seeds/api-product.sql:
//   Product A  → 11111111-1111-1111-1111-000000000001  (in stock, active)
//   Product B  → 11111111-1111-1111-1111-000000000002  (no stock, active)
//   Product C  → 11111111-1111-1111-1111-000000000003  (discontinued)

db = db.getSiblingDB('cart_db');

const now = new Date();

db.carts.insertMany([
  // ── customer_1: normal cart with two active products ──────────────────────
  {
    userId:    '22222222-2222-2222-2222-000000000001',
    items: [
      {
        itemId:    '11111111-1111-1111-1111-000000000001',
        productId: '11111111-1111-1111-1111-000000000001',
        name:      'Product A',
        price:     200.00,
        quantity:  2,
        addedAt:   now,
      },
      {
        itemId:    '11111111-1111-1111-1111-000000000002',
        productId: '11111111-1111-1111-1111-000000000002',
        name:      'Product B',
        price:     400.00,
        quantity:  1,
        addedAt:   now,
      },
    ],
    createdAt: now,
    updatedAt: now,
  },

  // ── customer_2: cart with a discontinued product — for error simulation ───
  {
    userId:    '22222222-2222-2222-2222-000000000002',
    items: [
      {
        itemId:    '11111111-1111-1111-1111-000000000003',
        productId: '11111111-1111-1111-1111-000000000003',
        name:      'Product C',
        price:     500.00,
        quantity:  1,
        addedAt:   now,
      },
    ],
    createdAt: now,
    updatedAt: now,
  },

  // ── ghost cart: userId has no matching customer — for error simulation ─────
  {
    userId:    '99999999-9999-9999-9999-999999999999',
    items: [
      {
        itemId:    '11111111-1111-1111-1111-000000000001',
        productId: '11111111-1111-1111-1111-000000000001',
        name:      'Product A',
        price:     200.00,
        quantity:  3,
        addedAt:   now,
      },
    ],
    createdAt: now,
    updatedAt: now,
  },
]);

print('api-cart seed complete: 3 carts inserted');
