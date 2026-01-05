# Homework 8 API规范 - 统一版本 (修订版)

## ⚠️ 重要说明
本文档为团队统一API和测试规范，确保MySQL和DynamoDB实现的一致性。其中部分规范（标记为"团队约定"）是对作业要求的必要补充。

---

## API接口定义

### ⚠️ 关于GET /shopping-carts/{id}
**注意：** OpenAPI规范中未定义此endpoint的响应格式，但作业明确要求实现。以下响应格式为**团队约定**。

### 1. POST /shopping-carts
创建购物车

**Request:**
```json
{
  "customer_id": 12345
}
```

**Response (201):**
```json
{
  "shopping_cart_id": 67890
}
```

**注意：** shopping_cart_id为**Integer类型**（纯数字，无引号）

---

### 2. POST /shopping-carts/{shoppingCartId}/items
添加商品到购物车

**Request:**
```json
{
  "product_id": 100,
  "quantity": 2
}
```

**Response:** 204 No Content (空body)

### ⚠️ 团队约定：Item更新行为
当添加已存在的product_id时：
- **✅ 覆盖quantity值**（例如：原2个，新传3个，结果为3个）
- **❌ 不是累加**（不是2+3=5）
- quantity必须大于0

---

### 3. GET /shopping-carts/{shoppingCartId}
获取购物车详情（**团队约定格式**）

**Response (200):**
```json
{
  "shopping_cart_id": 67890,
  "customer_id": 12345,
  "items": [
    {"product_id": 100, "quantity": 2}
  ]
}
```

**注意：**
- 空购物车返回 `items: []`
- 购物车不存在返回 404
- **此响应格式为团队约定**（OpenAPI未定义）

---

## 数据存储策略

**购物车只存储商品引用：**
- ✅ 存储: product_id + quantity
- ❌ 不存储: sku, manufacturer, weight等产品详细信息

**理由：** 简化设计，专注于数据库性能对比

---

## 错误响应格式

所有endpoints的错误响应：
```json
{
  "error": "ERROR_CODE",
  "message": "Human readable message"
}
```

**HTTP状态码：**
- `400` Bad Request: 输入数据无效
- `404` Not Found: 购物车或商品不存在
- `500` Internal Server Error: 服务器错误

---

## 数据库Schema建议

### MySQL (成员A)

**表1: shopping_carts**
```sql
CREATE TABLE shopping_carts (
    shopping_cart_id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_customer_id (customer_id)
) ENGINE=InnoDB;
```

**表2: cart_items**
```sql
CREATE TABLE cart_items (
    cart_item_id INT AUTO_INCREMENT PRIMARY KEY,
    shopping_cart_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (shopping_cart_id) 
        REFERENCES shopping_carts(shopping_cart_id) 
        ON DELETE CASCADE,
    UNIQUE KEY unique_cart_product (shopping_cart_id, product_id),
    INDEX idx_shopping_cart_id (shopping_cart_id)
) ENGINE=InnoDB;
```

**关键点：**
- 两张表，外键关联
- **UNIQUE KEY实现"覆盖更新"逻辑**（INSERT ... ON DUPLICATE KEY UPDATE）
- 索引优化查询性能

---

### DynamoDB (成员B)

**表名:** ShoppingCarts

**Partition Key:** shopping_cart_id (类型: **N** - Number)

**Item结构:**
```json
{
  "shopping_cart_id": 67890,
  "customer_id": 12345,
  "items": [
    {"product_id": 100, "quantity": 2},
    {"product_id": 101, "quantity": 3}
  ],
  "created_at": "2025-01-19T10:00:00Z",
  "updated_at": "2025-01-19T10:05:00Z"
}
```

**Terraform示例:**
```hcl
resource "aws_dynamodb_table" "shopping_carts" {
  name           = "ShoppingCarts"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "shopping_cart_id"

  attribute {
    name = "shopping_cart_id"
    type = "N"
  }
}
```

**关键点：**
- 单表设计
- shopping_cart_id生成: `Date.now()` 或 `Math.floor(Math.random() * 1000000)`
- items存为数组
- **添加item时检查数组中是否存在该product_id，存在则覆盖**

---

## 测试要求

### 测试数据生成规则

第 i 个购物车 (i = 0 to 49):
```
customer_id = 1000 + i
product_id  = 100 + (i % 10)  // 10个产品循环
quantity    = 1 + (i % 5)     // 1-5循环
```

**示例：**
- 购物车0: customer_id=1000, product_id=100, quantity=1
- 购物车1: customer_id=1001, product_id=101, quantity=2
- 购物车10: customer_id=1010, product_id=100, quantity=1

---

### 测试顺序

1. **Phase 1:** 创建50个购物车 (`POST /shopping-carts`)
2. **Phase 2:** 向50个购物车各添加1个商品 (`POST /shopping-carts/{id}/items`)
3. **Phase 3:** 获取50个购物车详情 (`GET /shopping-carts/{id}`)

**总计:** 150个操作，5分钟内完成

---

### 输出文件格式

#### 单独测试文件
**文件名：**
- `mysql_test_results.json`
- `dynamodb_test_results.json`

**JSON格式：**
```json
[
  {
    "operation": "create_cart",
    "response_time": 45.5,
    "success": true,
    "status_code": 201,
    "timestamp": "2025-01-19T10:00:00.123Z"
  }
]
```

**字段说明：**
- `operation`: `"create_cart"` | `"add_items"` | `"get_cart"`
- `response_time`: 毫秒数，保留1位小数
- `success`: boolean (status_code < 400)
- `status_code`: 201/204/200
- `timestamp`: ISO 8601格式

#### STEP III 合并文件格式
**文件名：** `combined_results.json`

**格式（成员C负责合并）：**
```json
[
  {
    "database": "mysql",
    "operation": "create_cart",
    "response_time": 45.5,
    "success": true,
    "status_code": 201,
    "timestamp": "2025-01-19T10:00:00.123Z"
  },
  {
    "database": "dynamodb",
    "operation": "create_cart",
    "response_time": 42.3,
    "success": true,
    "status_code": 201,
    "timestamp": "2025-01-19T10:00:01.456Z"
  }
  // ... 总共300条记录（150 MySQL + 150 DynamoDB）
]
```

**合并要求：**
- 添加 `"database"` 字段标识数据来源
- 保持原始数据不变
- 用于STEP III的对比分析

---

## ⚠️ 关键协调点

### 1. API必须完全一致

**A和B的API响应必须字节级一致：**
- ✅ 字段名: `shopping_cart_id` (不是 `cart_id`)
- ✅ 数据类型: Integer (纯数字，无引号)
- ✅ 状态码: 201/204/200
- ✅ 错误格式: 相同
- ✅ Item更新逻辑: 覆盖而非累加

---

### 2. 测试脚本统一

- 成员C提供统一测试脚本
- 成员A和B **只改BASE_URL**
- 使用完全相同的测试数据生成逻辑
- 确保timestamp格式一致（ISO 8601）

---

### 3. 时间线检查点

**Day 1 (今天):**
- A: Terraform + RDS MySQL
- B: Terraform + DynamoDB
- C: 测试脚本准备

**Day 2:**
- A和B: API实现完成
- **互相测试对方API**（Postman/curl）
- 验证响应格式一致
- 确认Item更新逻辑一致

**Day 3 上午:**
- A和B: 运行测试，生成JSON文件
- 验证格式一致性
- C: 准备合并脚本

**Day 3 下午 - Day 5:**
- C: 生成combined_results.json
- C: 分析和报告
- 整合提交

---

## 验证清单（Day 3上午）

### API一致性验证
- [ ] 两个API的shopping_cart_id都是Integer
- [ ] GET响应格式完全相同（团队约定格式）
- [ ] 状态码一致 (201/204/200)
- [ ] 错误格式一致
- [ ] Item更新行为一致（覆盖逻辑）

### 测试数据验证
- [ ] mysql_test_results.json 有150条记录
- [ ] dynamodb_test_results.json 有150条记录
- [ ] 操作分布: 50 create + 50 add + 50 get
- [ ] 所有字段名完全相同
- [ ] timestamp都是ISO 8601格式
- [ ] response_time单位是毫秒

### STEP III准备
- [ ] 准备好合并脚本
- [ ] 确认combined_results.json格式
- [ ] 准备性能分析工具

**如有不一致，立即在群里沟通修正！**

---

## 快速参考

### API
```
POST   /shopping-carts              → 201 {"shopping_cart_id": 67890}
POST   /shopping-carts/{id}/items   → 204 (empty)
GET    /shopping-carts/{id}         → 200 {shopping_cart_id, customer_id, items:[]}
```

### 状态码
```
201 - Cart created
204 - Item added (or updated)
200 - Cart retrieved
400 - Bad request
404 - Not found
500 - Server error
```

### 测试数据
```
customer_id: 1000-1049
product_id:  100-109 (循环)
quantity:    1-5 (循环)
Total:       150 operations
```

### 数据类型（重要！）
```
shopping_cart_id: Integer (not string!)
customer_id:      Integer
product_id:       Integer
quantity:         Integer
```

### 团队约定总结
1. **GET响应格式**：如上定义（OpenAPI未规定）
2. **Item更新**：覆盖而非累加
3. **ID生成**：MySQL用AUTO_INCREMENT，DynamoDB用时间戳或随机数

---

## 常见问题FAQ

**Q: GET /shopping-carts/{id} 的响应格式从哪来的？**
A: OpenAPI未定义，这是团队约定的格式，确保MySQL和DynamoDB返回相同结构。

**Q: 为什么选择覆盖而不是累加quantity？**
A: 作业未明确规定，团队选择覆盖逻辑更符合常见购物车行为（用户设置确切数量）。

**Q: shopping_cart_id必须是连续的吗？**
A: 不需要。MySQL可能是连续的，DynamoDB使用时间戳不会连续，这不影响测试。

**Q: 如何处理并发测试？**
A: 测试脚本应该是顺序执行的，避免并发问题影响性能对比的公平性。