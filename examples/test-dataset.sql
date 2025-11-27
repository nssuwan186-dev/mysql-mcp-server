-- MySQL MCP Server - Example Test Dataset
-- This creates a demo database for testing the MCP server

-- Create database
CREATE DATABASE IF NOT EXISTS mcp_demo;
USE mcp_demo;

-- Drop existing tables if they exist
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS customers;
DROP VIEW IF EXISTS order_summary;
DROP VIEW IF EXISTS product_inventory;

-- Create categories table
CREATE TABLE categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name (name)
) ENGINE=InnoDB COMMENT='Product categories';

-- Create products table
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    category_id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    stock_quantity INT DEFAULT 0,
    sku VARCHAR(50) UNIQUE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_category (category_id),
    INDEX idx_sku (sku),
    INDEX idx_price (price),
    CONSTRAINT fk_product_category FOREIGN KEY (category_id) REFERENCES categories(id)
) ENGINE=InnoDB COMMENT='Product catalog';

-- Create customers table
CREATE TABLE customers (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    address TEXT,
    city VARCHAR(100),
    country VARCHAR(100) DEFAULT 'USA',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP NULL,
    INDEX idx_email (email),
    INDEX idx_country (country),
    FULLTEXT INDEX ft_name (first_name, last_name)
) ENGINE=InnoDB COMMENT='Customer information';

-- Create orders table
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status ENUM('pending', 'confirmed', 'shipped', 'delivered', 'cancelled') DEFAULT 'pending',
    total_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
    shipping_address TEXT,
    notes TEXT,
    INDEX idx_customer (customer_id),
    INDEX idx_status (status),
    INDEX idx_order_date (order_date),
    CONSTRAINT fk_order_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
) ENGINE=InnoDB COMMENT='Customer orders';

-- Create order_items table
CREATE TABLE order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    unit_price DECIMAL(10,2) NOT NULL,
    total_price DECIMAL(10,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    INDEX idx_order (order_id),
    INDEX idx_product (product_id),
    CONSTRAINT fk_item_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_item_product FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB COMMENT='Order line items';

-- Insert sample categories
INSERT INTO categories (name, description) VALUES
('Electronics', 'Electronic devices and accessories'),
('Books', 'Physical and digital books'),
('Clothing', 'Apparel and fashion items'),
('Home & Garden', 'Home improvement and garden supplies'),
('Sports', 'Sports equipment and accessories');

-- Insert sample products
INSERT INTO products (category_id, name, description, price, stock_quantity, sku) VALUES
(1, 'Wireless Mouse', 'Ergonomic wireless mouse with USB receiver', 29.99, 150, 'ELEC-001'),
(1, 'Mechanical Keyboard', 'RGB mechanical keyboard with Cherry MX switches', 149.99, 75, 'ELEC-002'),
(1, 'USB-C Hub', '7-in-1 USB-C hub with HDMI, USB-A, and SD card', 59.99, 200, 'ELEC-003'),
(1, 'Webcam HD', '1080p webcam with built-in microphone', 79.99, 100, 'ELEC-004'),
(2, 'SQL Mastery', 'Complete guide to SQL databases', 49.99, 500, 'BOOK-001'),
(2, 'Go Programming', 'Learning Go programming language', 39.99, 300, 'BOOK-002'),
(2, 'Cloud Architecture', 'Designing scalable cloud systems', 59.99, 200, 'BOOK-003'),
(3, 'Cotton T-Shirt', 'Premium cotton t-shirt, various colors', 24.99, 1000, 'CLOTH-001'),
(3, 'Denim Jeans', 'Classic fit denim jeans', 79.99, 500, 'CLOTH-002'),
(4, 'Garden Tools Set', '5-piece garden tool set', 34.99, 150, 'HOME-001'),
(4, 'LED Desk Lamp', 'Adjustable LED desk lamp with USB charging', 44.99, 300, 'HOME-002'),
(5, 'Yoga Mat', 'Non-slip yoga mat, 6mm thick', 29.99, 400, 'SPORT-001'),
(5, 'Resistance Bands', 'Set of 5 resistance bands', 19.99, 600, 'SPORT-002');

-- Insert sample customers
INSERT INTO customers (email, first_name, last_name, phone, city, country) VALUES
('john.doe@example.com', 'John', 'Doe', '+1-555-0101', 'New York', 'USA'),
('jane.smith@example.com', 'Jane', 'Smith', '+1-555-0102', 'Los Angeles', 'USA'),
('bob.wilson@example.com', 'Bob', 'Wilson', '+1-555-0103', 'Chicago', 'USA'),
('alice.johnson@example.com', 'Alice', 'Johnson', '+44-20-7946-0958', 'London', 'UK'),
('carlos.garcia@example.com', 'Carlos', 'Garcia', '+34-91-123-4567', 'Madrid', 'Spain'),
('yuki.tanaka@example.com', 'Yuki', 'Tanaka', '+81-3-1234-5678', 'Tokyo', 'Japan'),
('maria.silva@example.com', 'Maria', 'Silva', '+55-11-98765-4321', 'São Paulo', 'Brazil'),
('hans.mueller@example.com', 'Hans', 'Mueller', '+49-30-12345678', 'Berlin', 'Germany');

-- Insert sample orders
INSERT INTO orders (customer_id, status, total_amount, shipping_address) VALUES
(1, 'delivered', 209.97, '123 Main St, New York, NY 10001'),
(1, 'shipped', 149.99, '123 Main St, New York, NY 10001'),
(2, 'confirmed', 89.98, '456 Oak Ave, Los Angeles, CA 90001'),
(3, 'pending', 174.97, '789 Pine Rd, Chicago, IL 60601'),
(4, 'delivered', 59.99, '10 Downing St, London, UK'),
(5, 'shipped', 129.98, 'Calle Mayor 1, Madrid, Spain'),
(6, 'delivered', 49.99, '1-2-3 Shibuya, Tokyo, Japan'),
(7, 'cancelled', 79.99, 'Av Paulista 1000, São Paulo, Brazil'),
(8, 'pending', 184.97, 'Unter den Linden 1, Berlin, Germany');

-- Insert sample order items
INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
(1, 1, 2, 29.99),
(1, 2, 1, 149.99),
(2, 2, 1, 149.99),
(3, 3, 1, 59.99),
(3, 1, 1, 29.99),
(4, 5, 2, 49.99),
(4, 6, 1, 39.99),
(4, 7, 1, 59.99),
(5, 3, 1, 59.99),
(6, 8, 2, 24.99),
(6, 9, 1, 79.99),
(7, 5, 1, 49.99),
(8, 9, 1, 79.99),
(9, 10, 1, 34.99),
(9, 11, 2, 44.99),
(9, 12, 2, 29.99);

-- Create views
CREATE VIEW order_summary AS
SELECT 
    o.id AS order_id,
    c.email AS customer_email,
    CONCAT(c.first_name, ' ', c.last_name) AS customer_name,
    o.order_date,
    o.status,
    o.total_amount,
    COUNT(oi.id) AS item_count
FROM orders o
JOIN customers c ON o.customer_id = c.id
LEFT JOIN order_items oi ON o.id = oi.order_id
GROUP BY o.id, c.email, c.first_name, c.last_name, o.order_date, o.status, o.total_amount;

CREATE VIEW product_inventory AS
SELECT 
    p.id,
    p.name AS product_name,
    c.name AS category_name,
    p.price,
    p.stock_quantity,
    p.sku,
    CASE 
        WHEN p.stock_quantity = 0 THEN 'Out of Stock'
        WHEN p.stock_quantity < 50 THEN 'Low Stock'
        ELSE 'In Stock'
    END AS stock_status
FROM products p
JOIN categories c ON p.category_id = c.id
WHERE p.is_active = TRUE;

-- Create a stored procedure (for extended tools testing)
DELIMITER //
CREATE PROCEDURE GetCustomerOrders(IN customer_email VARCHAR(255))
BEGIN
    SELECT o.id, o.order_date, o.status, o.total_amount
    FROM orders o
    JOIN customers c ON o.customer_id = c.id
    WHERE c.email = customer_email
    ORDER BY o.order_date DESC;
END //
DELIMITER ;

-- Create a stored function
DELIMITER //
CREATE FUNCTION GetProductStock(product_sku VARCHAR(50))
RETURNS INT
DETERMINISTIC
READS SQL DATA
BEGIN
    DECLARE stock INT;
    SELECT stock_quantity INTO stock FROM products WHERE sku = product_sku;
    RETURN IFNULL(stock, 0);
END //
DELIMITER ;

-- Summary message
SELECT 'MCP Demo Database Created Successfully!' AS message;
SELECT 
    (SELECT COUNT(*) FROM categories) AS categories,
    (SELECT COUNT(*) FROM products) AS products,
    (SELECT COUNT(*) FROM customers) AS customers,
    (SELECT COUNT(*) FROM orders) AS orders,
    (SELECT COUNT(*) FROM order_items) AS order_items;

