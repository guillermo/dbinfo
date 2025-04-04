-- Clean up if tables already exist
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS customers;

-- Create tables with various features to test

-- Categories table
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE categories IS 'Product categories';
COMMENT ON COLUMN categories.name IS 'Category name';

-- Products table with foreign key
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    description TEXT,
    sku VARCHAR(50) UNIQUE,
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

COMMENT ON TABLE products IS 'Available products';

-- Create indexes on products
CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_name ON products(name);
CREATE UNIQUE INDEX idx_products_sku ON products(sku);

-- Customers table
CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    address TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE customers IS 'Customer information';

-- Orders table with foreign key
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    order_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(10, 2) NOT NULL,
    shipping_address TEXT,
    tracking_number VARCHAR(100),
    notes TEXT
);

COMMENT ON TABLE orders IS 'Customer orders';

-- Create index on orders
CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_date ON orders(order_date);

-- Order items table with multiple foreign keys
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10, 2) NOT NULL,
    subtotal DECIMAL(10, 2) NOT NULL,
    UNIQUE (order_id, product_id)
);

COMMENT ON TABLE order_items IS 'Individual items within an order';

-- Create indexes on order_items
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);

-- Insert some sample data

-- Categories
INSERT INTO categories (name, description) VALUES
('Electronics', 'Electronic devices and gadgets'),
('Clothing', 'Apparel and accessories'),
('Books', 'Books and publications');

-- Products
INSERT INTO products (category_id, name, price, description, sku, stock_quantity) VALUES
(1, 'Smartphone', 699.99, 'Latest model smartphone', 'PHONE-001', 50),
(1, 'Laptop', 1299.99, 'High-performance laptop', 'LAPTOP-001', 25),
(2, 'T-Shirt', 19.99, 'Cotton t-shirt', 'TSHIRT-001', 100),
(3, 'Novel', 14.99, 'Bestselling fiction novel', 'BOOK-001', 75);

-- Customers
INSERT INTO customers (email, first_name, last_name, phone, address) VALUES
('john.doe@example.com', 'John', 'Doe', '555-1234', '123 Main St, Anytown, USA'),
('jane.smith@example.com', 'Jane', 'Smith', '555-5678', '456 Oak Ave, Somewhere, USA');

-- Orders
INSERT INTO orders (customer_id, status, total_amount, shipping_address) VALUES
(1, 'completed', 714.98, '123 Main St, Anytown, USA'),
(2, 'processing', 1319.98, '456 Oak Ave, Somewhere, USA');

-- Order items
INSERT INTO order_items (order_id, product_id, quantity, unit_price, subtotal) VALUES
(1, 1, 1, 699.99, 699.99),
(1, 3, 1, 14.99, 14.99),
(2, 2, 1, 1299.99, 1299.99),
(2, 3, 1, 19.99, 19.99);