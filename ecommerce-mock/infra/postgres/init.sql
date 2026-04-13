-- PostgreSQL init script
-- Creates the three databases needed by Go services
-- product_db is already created via POSTGRES_DB env var

CREATE DATABASE customer_db
    WITH OWNER = ecommerce
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TEMPLATE = template0;

CREATE DATABASE order_db
    WITH OWNER = ecommerce
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.utf8'
    LC_CTYPE = 'en_US.utf8'
    TEMPLATE = template0;

-- Grant all privileges to ecommerce user
GRANT ALL PRIVILEGES ON DATABASE product_db TO ecommerce;
GRANT ALL PRIVILEGES ON DATABASE customer_db TO ecommerce;
GRANT ALL PRIVILEGES ON DATABASE order_db TO ecommerce;
