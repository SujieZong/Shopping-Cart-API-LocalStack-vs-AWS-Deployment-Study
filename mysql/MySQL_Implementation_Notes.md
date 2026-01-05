# STEP I: MySQL Implementation Notes

## Database Schema Design Decisions
- **Two-table design**: shopping_carts (parent) and cart_items (child) with foreign key relationship
- **Indexing strategy**: Indexes on customer_id and shopping_cart_id for fast lookups
- **AUTO_INCREMENT**: Used for cart IDs to ensure uniqueness
- **ON DELETE CASCADE**: Automatic cleanup of items when cart is deleted
- **UNIQUE constraint**: (shopping_cart_id, product_id) prevents duplicate items
- **Transaction handling**: Used ON DUPLICATE KEY UPDATE for atomic item updates

## Key Challenges & Solutions
- **Challenge**: RDS in private subnet couldn't be accessed directly
- **Solution**: Implemented initSchema() function to auto-create tables on startup
- **Challenge**: ECS tasks kept restarting
- **Solution**: Removed task-level health check in main.tf
- **Challenge**: Race condition in test script causing incomplete results
- **Solution**: Discovered probabilistic nature of concurrency issues; mutex protection added for production
- **Challenge**: Initial schema without indexes was slow
- **Solution**: Added indexes on foreign keys, reduced query time by 60%

## Performance Observations
- **Connection Pool Config**: MaxOpenConns=10, MaxIdleConns=5 (balanced for moderate load)
- **Test Results**: 100% success rate (150/150 operations)
  - Overall: avg 42.6ms (P50: 30ms, P95: 106ms, P99: 145ms)
  - create_cart: avg 31.8ms (P50: 21.2ms, P95: 82.6ms)
  - add_items: avg 32.3ms (P50: 23.7ms, P95: 97.6ms)
  - get_cart: avg 63.6ms (P50: 48.1ms, P95: 136.8ms)
- **Resource Usage**: CPU peaked at 6%, 8 DB connections handled 150 requests efficiently
- **Total Test Duration**: 2.05 seconds for all 150 operations
- **Bottleneck**: get_cart operations (63.6ms avg) are 2x slower than writes due to JOIN between shopping_carts and cart_items tables

## Comparison with Week 5 In-Memory Approach
| Metric | HW5 In-Memory | HW8 MySQL | Impact |
|--------|---------------|-----------|---------|
| Average Response | 26ms | 42.6ms | +64% latency |
| P99 Latency | 58ms | 145ms | +150% tail latency |
| Throughput | 156 RPS @200users | ~73 RPS | -53% RPS |
| Scalability | Linear to 200+ | Limited by pool | Connection bound |
| Data Durability | None | Full ACID | ✓ Persistent |
| Complexity | Simple | Schema + Pool + TX | Higher operational overhead |

### Trade-off Analysis
- **Performance Impact**: Significant P99 degradation (58ms → 145ms) shows database overhead
- **Consistency**: Strong ACID guarantees vs in-memory eventual consistency risks
- **Operational Complexity**: Required connection pooling, transaction management, and schema design
- **Scalability Path**: MySQL can add read replicas; in-memory limited by single instance RAM

## Key Learnings
- **Schema Evolution**: Initial design needed optimization; indexes were critical for performance
- **Connection Pooling**: Essential for production to prevent connection exhaustion
- **Concurrency Insights**: Discovered race conditions are probabilistic; proper synchronization critical even for testing tools
- **Read vs Write Performance**: JOINs in get_cart caused 2x latency compared to simple inserts
- **Future Optimization**: Would consider read replicas for scaling read-heavy operations and query caching for frequently accessed carts