// MongoDB init script for cart_db
// Runs as root after MONGO_INITDB_DATABASE is created

db = db.getSiblingDB('cart_db');

db.createUser({
  user: 'ecommerce',
  pwd: 'ecommerce_pass',
  roles: [{ role: 'readWrite', db: 'cart_db' }]
});

// carts collection with TTL index (24h cart expiry)
db.createCollection('carts');
db.carts.createIndex({ updatedAt: 1 }, { expireAfterSeconds: 86400, name: 'cart_ttl_idx' });
db.carts.createIndex({ userId: 1 }, { unique: true, name: 'cart_user_idx' });
