-- db-setup.sql
-- 数据库Schema初始化脚本

-- 创建数据库（如果需要）
CREATE DATABASE IF NOT EXISTS shopping_cart_db;
USE shopping_cart_db;

-- 删除旧表（如果存在）
DROP TABLE IF EXISTS cart_items;
DROP TABLE IF EXISTS shopping_carts;

-- 创建shopping_carts表
CREATE TABLE shopping_carts (
    shopping_cart_id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_customer_id (customer_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 创建cart_items表
CREATE TABLE cart_items (
    cart_item_id INT AUTO_INCREMENT PRIMARY KEY,
    shopping_cart_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (shopping_cart_id) 
        REFERENCES shopping_carts(shopping_cart_id) 
        ON DELETE CASCADE,
    UNIQUE KEY unique_cart_product (shopping_cart_id, product_id),
    INDEX idx_shopping_cart_id (shopping_cart_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 验证表结构
SHOW TABLES;
DESCRIBE shopping_carts;
DESCRIBE cart_items;

-- 测试查询性能
EXPLAIN SELECT 
    c.shopping_cart_id,
    c.customer_id,
    ci.product_id,
    ci.quantity
FROM shopping_carts c
LEFT JOIN cart_items ci ON c.shopping_cart_id = ci.shopping_cart_id
WHERE c.shopping_cart_id = 1;