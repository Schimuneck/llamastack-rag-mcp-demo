# Terminate active connections to the database
psql -d postgres -c "SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = 'currys_salesforce' AND pid <> pg_backend_pid();"

# Drop the database
psql -d postgres -c "DROP DATABASE IF EXISTS currys_salesforce;"
psql -d postgres -c "CREATE DATABASE currys_salesforce;"

psql -d currys_salesforce <<'SQL'
-- Drop tables in correct order (child tables first)
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS customers CASCADE;

CREATE TABLE customers (
  customer_id       SERIAL PRIMARY KEY,
  first_name        text NOT NULL,
  last_name         text NOT NULL,
  address           text NOT NULL,
  city              text NOT NULL,
  country           text NOT NULL,
  phone_number      text NOT NULL UNIQUE,
  email             text NOT NULL UNIQUE
);

CREATE TABLE products (
  product_id        SERIAL PRIMARY KEY,
  product_name      text NOT NULL UNIQUE,
  price             NUMERIC(10,2),
  category          text NOT NULL
);

CREATE TABLE orders (
  order_id              SERIAL PRIMARY KEY,
  customer_id           integer NOT NULL,
  product_id            integer NOT NULL,
  order_date            timestamp NOT NULL,
  FOREIGN KEY (customer_id) REFERENCES customers(customer_id),
  FOREIGN KEY (product_id) REFERENCES products(product_id)
);

-- Insert test customer
INSERT INTO customers (first_name, last_name, address, city, country, phone_number, email) 
VALUES ('John', 'Doe', '123 Main Street', 'San Francisco', 'USA', '+1-555-0123', 'john.doe@example.com');

-- Insert test products
INSERT INTO products (product_name, price, category) VALUES
('Wireless Logitech Keyboard', 79.99, 'Computer Accessories'),
('MacBook Pro', 1999.99, 'Laptops'),
('Wireless Logitech Mouse', 39.99, 'Computer Accessories'),
('MSI Monitor', 134.00, 'Monitors');
SQL
