# STEP III: Database Comparison & Analysis

## Part 0: Verification
MySQL 150 records, DynamoDB 150 records ✅  
Data file: `combined_results.json`

## Part 1: Performance Comparison

| Metric | MySQL | DynamoDB | Winner | Margin |
|--|--:|--:|--:|--:|
| Avg Response Time (ms) | 42.59 | 38.03 | DynamoDB | 4.56 |
| P50 Response Time (ms) | 29.05 | 30.74 | MySQL | 1.70 |
| P95 Response Time (ms) | 123.17 | 72.28 | DynamoDB | 50.89 |
| P99 Response Time (ms) | 141.68 | 120.21 | DynamoDB | 21.47 |
| Success Rate (%) | 100.00 | 100.00 | Tie | 0.00 |

### Operation Breakdown

| Operation | MySQL Avg (ms) | DynamoDB Avg (ms) | Faster By |
|--|--:|--:|--:|
| CREATE_CART | 31.83 | 33.86 | 2.04 |
| ADD_ITEMS | 32.33 | 50.62 | 18.28 |
| GET_CART | 63.61 | 29.61 | 34.00 |
