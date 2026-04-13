-- MySQL init script for payment_db

USE payment_db;

-- Ensure ecommerce user has full access to payment_db
GRANT ALL PRIVILEGES ON payment_db.* TO 'ecommerce'@'%';
FLUSH PRIVILEGES;
