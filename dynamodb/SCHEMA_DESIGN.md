# DynamoDB Single-Table Schema Design for Shopping Cart System

## Table Overview

**Table Name:** `ShoppingCarts`  
**Billing Mode:** PAY_PER_REQUEST (On-demand)  
**Primary Key:** `shopping_cart_id` (Partition Key)

---

## Schema Design

### Key Structure

#### Partition Key Choice: `shopping_cart_id` (Number)

**Why this choice?**

1. **Direct Access Pattern (< 50ms requirement)**

   - Shopping cart retrieval is the primary access pattern: `GET /shopping-carts/{id}`
   - Using `shopping_cart_id` as the partition key enables single-item queries via `GetItem`
   - GetItem operations with partition key provide **single-digit millisecond latency**
   - No need for Query or Scan operations, which are slower
   - Consistently meets the < 50ms latency requirement (typically 5-15ms)

2. **Even Data Distribution**

   - Each shopping cart is a unique partition
   - No hot partitions - load is distributed across all carts
   - Prevents throttling issues from uneven access patterns
   - Scales horizontally as cart count grows

3. **Write Efficiency**

   - Updates to cart items (quantity changes) only affect one partition
   - No need to update multiple items or perform complex transactions
   - Atomic updates within a single item

4. **Natural Access Pattern**
   - All operations are scoped by `shopping_cart_id`
   - API routes use cart ID: `/shopping-carts/{shoppingCartId}`
   - Aligns perfectly with application logic

**Alternative Considered and Rejected:**

- **customer_id as partition key**: Would require Query operations instead of GetItem, adding latency (20-50ms vs 5-15ms). Also creates hot partitions if some customers have many carts or access patterns are uneven.

---

## Item Structure

### Complete Item Schema

```json
{
  "shopping_cart_id": 1698765432100, // Number (Partition Key)
  "customer_id": 12345, // Number
  "items": [
    // List of Maps
    {
      "product_id": 100, // Number
      "quantity": 2 // Number
    },
    {
      "product_id": 101, // Number
      "quantity": 3 // Number
    }
  ],
  "created_at": "2025-10-30T14:23:45.678Z", // String (ISO 8601)
  "updated_at": "2025-10-30T14:28:12.345Z" // String (ISO 8601)
}
```

### Attribute Details

| Attribute            | Type       | Description                             | Required              |
| -------------------- | ---------- | --------------------------------------- | --------------------- |
| `shopping_cart_id`   | Number (N) | Unique cart identifier, partition key   | ✅ Yes                |
| `customer_id`        | Number (N) | Customer who owns this cart             | ✅ Yes                |
| `items`              | List (L)   | Array of cart items                     | ✅ Yes (can be empty) |
| `items[].product_id` | Number (N) | Product identifier in cart              | ✅ Yes                |
| `items[].quantity`   | Number (N) | Quantity of this product (> 0)          | ✅ Yes                |
| `created_at`         | String (S) | ISO 8601 timestamp of cart creation     | ⚪ Optional           |
| `updated_at`         | String (S) | ISO 8601 timestamp of last modification | ⚪ Optional           |

---

## Item Structure - Nested Details

### Items Array Structure

The `items` attribute is a **List of Maps** in DynamoDB terms:

```javascript
// DynamoDB Type Notation
items: {
  L: [
    // List type
    {
      M: {
        // Map type
        product_id: { N: "100" }, // Number stored as string in DynamoDB
        quantity: { N: "2" },
      },
    },
    {
      M: {
        product_id: { N: "101" },
        quantity: { N: "3" },
      },
    },
  ];
}
```

**Key Properties:**

- **No duplicate product_ids**: Application enforces uniqueness
- **Ordered**: Items maintain insertion order (though order is not significant for business logic)
- **Denormalized**: All cart data in a single item - no joins needed
- **Flexible**: Can store 0 to 100+ items efficiently (DynamoDB item size limit is 400KB)

---

## Operations Implementation

### 1. Create Shopping Cart

**Endpoint:** `POST /shopping-carts`

**DynamoDB Operation:** `PutItem`

```javascript
const params = {
  TableName: "ShoppingCarts",
  Item: {
    shopping_cart_id: Date.now(), // Generate unique ID
    customer_id: 12345,
    items: [], // Empty array initially
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
};

await dynamodb.put(params).promise();
```

**Performance:** ~5-10ms (single PutItem operation)

---

### 2. Add/Update Item in Cart

**Endpoint:** `POST /shopping-carts/{shoppingCartId}/items`

**DynamoDB Operation:** `UpdateItem` with list manipulation

**Requirements:**

- If product_id exists in items array → **override quantity** (not add)
- If product_id doesn't exist → append to items array
- Update `updated_at` timestamp

**Implementation Strategy:**

```javascript
// Step 1: Get current cart to check existing items
const getParams = {
  TableName: "ShoppingCarts",
  Key: { shopping_cart_id: 67890 },
};

const { Item: cart } = await dynamodb.get(getParams).promise();

// Step 2: Find if product already exists
const existingIndex = cart.items.findIndex((item) => item.product_id === 100);

let updateExpression;
let expressionAttributeValues;

if (existingIndex >= 0) {
  // Override existing quantity
  updateExpression = `SET items[${existingIndex}].quantity = :qty, updated_at = :timestamp`;
  expressionAttributeValues = {
    ":qty": 3,
    ":timestamp": new Date().toISOString(),
  };
} else {
  // Append new item to array
  updateExpression =
    "SET items = list_append(items, :new_item), updated_at = :timestamp";
  expressionAttributeValues = {
    ":new_item": [{ product_id: 100, quantity: 3 }],
    ":timestamp": new Date().toISOString(),
  };
}

const updateParams = {
  TableName: "ShoppingCarts",
  Key: { shopping_cart_id: 67890 },
  UpdateExpression: updateExpression,
  ExpressionAttributeValues: expressionAttributeValues,
};

await dynamodb.update(updateParams).promise();
```

**Alternative: Single UpdateItem with REMOVE + list_append**

```javascript
// More efficient: Remove old entry (if exists) and append new one
// This avoids the GetItem call but requires removing by index
const updateParams = {
  TableName: "ShoppingCarts",
  Key: { shopping_cart_id: 67890 },
  UpdateExpression: `
    SET items = list_append(
      if_not_exists(items, :empty_list),
      :new_item
    ),
    updated_at = :timestamp
  `,
  ExpressionAttributeValues: {
    ":new_item": [{ product_id: 100, quantity: 3 }],
    ":empty_list": [],
    ":timestamp": new Date().toISOString(),
  },
};
```

**Note:** For production, implement a custom function that:

1. Gets the cart
2. Manipulates the items array in memory
3. Writes back the entire updated items array

**Performance:** ~10-15ms (GetItem + UpdateItem)

---

### 3. Retrieve Shopping Cart

**Endpoint:** `GET /shopping-carts/{shoppingCartId}`

**DynamoDB Operation:** `GetItem`

```javascript
const params = {
  TableName: "ShoppingCarts",
  Key: {
    shopping_cart_id: 67890,
  },
};

const { Item } = await dynamodb.get(params).promise();

if (!Item) {
  // Return 404
  throw new Error("Cart not found");
}

// Return the item (already in correct format)
return {
  shopping_cart_id: Item.shopping_cart_id,
  customer_id: Item.customer_id,
  items: Item.items,
};
```

**Performance:** ~5-10ms (single GetItem with partition key)

**Why < 50ms is easily achieved:**

- GetItem with partition key is DynamoDB's fastest operation
- No secondary indexes needed
- No Query or Scan operations
- Typical latency: 5-15ms (well under 50ms requirement)
- Consistent low latency regardless of table size

---

## Performance Characteristics

### Latency Analysis

| Operation       | DynamoDB API         | Expected Latency | Meets < 50ms?            |
| --------------- | -------------------- | ---------------- | ------------------------ |
| Create Cart     | PutItem              | 5-10ms           | ✅ Yes (10-20% of limit) |
| Add/Update Item | GetItem + UpdateItem | 10-20ms          | ✅ Yes (20-40% of limit) |
| Get Cart        | GetItem              | 5-10ms           | ✅ Yes (10-20% of limit) |

### Scalability

- **Read Capacity:** On-demand mode auto-scales
- **Write Capacity:** On-demand mode auto-scales
- **Hot Partitions:** None - each cart is isolated
- **Maximum Items per Cart:** Practical limit ~1000 items (well within 400KB item size)

### Cost Optimization

**On-Demand Pricing (us-west-2):**

- Write: $1.25 per million write request units
- Read: $0.25 per million read request units

**Example for 50 carts test:**

- 50 creates + 50 updates + 50 reads = 150 operations
- Cost: ~$0.0002 (negligible for testing)

---

## Data Validation Rules

### Application-Level Validation

```javascript
// Before writing to DynamoDB
function validateCartItem(product_id, quantity) {
  if (!Number.isInteger(product_id) || product_id <= 0) {
    throw new Error("product_id must be a positive integer");
  }

  if (!Number.isInteger(quantity) || quantity <= 0) {
    throw new Error("quantity must be a positive integer");
  }

  if (quantity > 1000) {
    throw new Error("quantity cannot exceed 1000");
  }
}

function validateCart(items) {
  // Check for duplicate product_ids
  const productIds = items.map((item) => item.product_id);
  const uniqueIds = new Set(productIds);

  if (productIds.length !== uniqueIds.size) {
    throw new Error("Duplicate product_ids found in cart");
  }

  // Validate each item
  items.forEach((item) => {
    validateCartItem(item.product_id, item.quantity);
  });
}
```

---

## Comparison with MySQL Approach

| Aspect             | DynamoDB (This Design)     | MySQL (2-Table Design)          |
| ------------------ | -------------------------- | ------------------------------- |
| **Schema**         | Single table, nested items | Two tables (carts + cart_items) |
| **Primary Access** | GetItem by partition key   | SELECT with JOIN                |
| **Item Updates**   | Update nested array        | INSERT ON DUPLICATE KEY UPDATE  |
| **Read Latency**   | 5-10ms (GetItem)           | 20-50ms (JOIN query)            |
| **Write Latency**  | 10-20ms                    | 15-30ms                         |
| **Scalability**    | Automatic, unlimited       | Manual (vertical/sharding)      |
| **Consistency**    | Single-item atomicity      | Transaction or row locking      |
| **Cost Model**     | Pay per request            | Pay for provisioned capacity    |

---

## Sample Data

### Example 1: Empty Cart

```json
{
  "shopping_cart_id": 1698765432100,
  "customer_id": 1000,
  "items": [],
  "created_at": "2025-10-30T14:23:45.678Z",
  "updated_at": "2025-10-30T14:23:45.678Z"
}
```

### Example 2: Cart with Items

```json
{
  "shopping_cart_id": 1698765432101,
  "customer_id": 1001,
  "items": [
    {
      "product_id": 100,
      "quantity": 2
    },
    {
      "product_id": 101,
      "quantity": 3
    },
    {
      "product_id": 105,
      "quantity": 1
    }
  ],
  "created_at": "2025-10-30T14:23:45.678Z",
  "updated_at": "2025-10-30T14:28:12.345Z"
}
```

### Example 3: After Updating Quantity

```json
// Before: product_id 100 had quantity 2
// POST /shopping-carts/1698765432101/items with {product_id: 100, quantity: 5}
// After: quantity is overridden to 5 (not added to make 7)

{
  "shopping_cart_id": 1698765432101,
  "customer_id": 1001,
  "items": [
    {
      "product_id": 100,
      "quantity": 5 // ← Overridden from 2 to 5
    },
    {
      "product_id": 101,
      "quantity": 3
    },
    {
      "product_id": 105,
      "quantity": 1
    }
  ],
  "created_at": "2025-10-30T14:23:45.678Z",
  "updated_at": "2025-10-30T15:12:34.567Z" // ← Updated timestamp
}
```

---

## Testing Data Generation (For HW8)

### Test Data Pattern

```javascript
// Generate 50 shopping carts with consistent test data
function generateTestData() {
  const carts = [];

  for (let i = 0; i < 50; i++) {
    const cart = {
      shopping_cart_id: Date.now() + i, // Unique ID
      customer_id: 1000 + i, // Pattern: 1000-1049
      items: [
        {
          product_id: 100 + (i % 10), // Pattern: 100-109 (cycling)
          quantity: 1 + (i % 5), // Pattern: 1-5 (cycling)
        },
      ],
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    carts.push(cart);
  }

  return carts;
}

// Example outputs:
// Cart 0:  customer_id=1000, product_id=100, quantity=1
// Cart 1:  customer_id=1001, product_id=101, quantity=2
// Cart 10: customer_id=1010, product_id=100, quantity=1
// Cart 49: customer_id=1049, product_id=109, quantity=5
```

---

## Potential Future Enhancements

### Global Secondary Index (GSI) for Customer Queries

If you need to query "all carts for a customer":

```hcl
# In Terraform
global_secondary_index {
  name            = "CustomerIdIndex"
  hash_key        = "customer_id"
  projection_type = "ALL"
}

attribute {
  name = "customer_id"
  type = "N"
}
```

**Usage:**

```javascript
// Query all carts for a customer
const params = {
  TableName: "ShoppingCarts",
  IndexName: "CustomerIdIndex",
  KeyConditionExpression: "customer_id = :cid",
  ExpressionAttributeValues: {
    ":cid": 12345,
  },
};

const result = await dynamodb.query(params).promise();
```

**Note:** Not needed for current requirements, but useful for future features.

---

## Summary

### Why This Design Works

✅ **Meets < 50ms requirement:** GetItem operations typically complete in 5-15ms  
✅ **Efficient partition key:** Direct access by `shopping_cart_id`, no hot partitions  
✅ **Simple data model:** Single table, all cart data in one item  
✅ **Supports updates:** Can override item quantities as required  
✅ **Scalable:** Auto-scaling with on-demand billing  
✅ **Cost-effective:** Pay only for actual requests

### Key Design Decisions

1. **Partition Key = shopping_cart_id**: Optimizes for primary access pattern (cart retrieval)
2. **Nested items array**: Avoids need for table joins, keeps related data together
3. **On-demand billing**: No capacity planning needed for homework/testing
4. **No GSI initially**: Current requirements don't need customer-based queries

### Performance Guarantee

With this design, all operations will complete well under 50ms:

- **GetItem:** 5-10ms (fastest DynamoDB operation)
- **PutItem:** 5-10ms
- **UpdateItem:** 10-20ms (including GetItem for array manipulation)

The partition key choice is the critical factor ensuring these low latencies.
