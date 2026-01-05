import json
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

# ---------- Input Files ----------
MYSQL_FILE = "mysql_test_results.json"
DYNAMO_FILE = "dynamodb_test_results.json"

# ---------- Output Files ----------
COMBINED_FILE = "combined_results.json"
REPORT_FILE = "step3_report.md"

# ---------- Helper Functions ----------
def percentile(series, q):
    """Return the q-th percentile of a numeric pandas Series."""
    return float(np.percentile(series, q))

def fmt(v):
    """Format a number with 2 decimal places."""
    return f"{v:.2f}"

# ---------- Read and Validate Data ----------
with open(MYSQL_FILE) as f1, open(DYNAMO_FILE) as f2:
    mysql = json.load(f1)
    dynamo = json.load(f2)

# Tag each record with its source database
for r in mysql:
    r["database"] = "MySQL"
for r in dynamo:
    r["database"] = "DynamoDB"

# Combine both datasets
combined = mysql + dynamo
df = pd.DataFrame(combined)

# Validate total record count (150 for each DB)
if len(df[df["database"] == "MySQL"]) != 150 or len(df[df["database"] == "DynamoDB"]) != 150:
    raise ValueError("❌ Data inconsistency: Each database must have 150 records (50 create, 50 add, 50 get)")

# Save the merged dataset
with open(COMBINED_FILE, "w") as f:
    json.dump(combined, f, indent=2)
print(f"✅ Combined file generated: {COMBINED_FILE}")

with open(COMBINED_FILE) as f:
    verify_data = json.load(f)
verify_df = pd.DataFrame(verify_data)

total = len(verify_df)
mysql_count = len(verify_df[verify_df["database"] == "MySQL"])
dynamo_count = len(verify_df[verify_df["database"] == "DynamoDB"])
mysql_ops = verify_df[verify_df["database"] == "MySQL"]["operation"].value_counts().to_dict()
dynamo_ops = verify_df[verify_df["database"] == "DynamoDB"]["operation"].value_counts().to_dict()

print("\n--- Data Consistency Check ---")
print(f"Total records: {total} (expected 300)")
print(f"MySQL records: {mysql_count} (expected 150)")
print(f"DynamoDB records: {dynamo_count} (expected 150)")
print(f"MySQL operations: {mysql_ops}")
print(f"DynamoDB operations: {dynamo_ops}")

if total != 300 or mysql_count != 150 or dynamo_count != 150:
    raise ValueError("❌ Combined file failed consistency check: Record count mismatch.")
for db_name, ops in [("MySQL", mysql_ops), ("DynamoDB", dynamo_ops)]:
    for op in ["create_cart", "add_items", "get_cart"]:
        if ops.get(op, 0) != 50:
            raise ValueError(f"❌ {db_name} missing or incorrect operation count for {op}. Expected 50, found {ops.get(op, 0)}.")

print("✅ Combined file verified successfully. Proceeding to analysis.\n")
# ---------- Compute Key Metrics ----------
mysql_series = df[df["database"] == "MySQL"]["response_time"]
dynamo_series = df[df["database"] == "DynamoDB"]["response_time"]

def row(metric, mysql_val, dynamo_val):
    """Return a row comparing MySQL vs DynamoDB for a given metric."""
    if mysql_val < dynamo_val:
        winner = "MySQL"
    elif dynamo_val < mysql_val:
        winner = "DynamoDB"
    else:
        winner = "Tie"
    return (metric, mysql_val, dynamo_val, winner, abs(mysql_val - dynamo_val))

# Build summary table
rows = [
    row("Avg Response Time (ms)", mysql_series.mean(), dynamo_series.mean()),
    row("P50 Response Time (ms)", percentile(mysql_series, 50), percentile(dynamo_series, 50)),
    row("P95 Response Time (ms)", percentile(mysql_series, 95), percentile(dynamo_series, 95)),
    row("P99 Response Time (ms)", percentile(mysql_series, 99), percentile(dynamo_series, 99)),
    row("Success Rate (%)",
        df[df["database"] == "MySQL"]["success"].mean() * 100,
        df[df["database"] == "DynamoDB"]["success"].mean() * 100),
]
overall = pd.DataFrame(rows, columns=["Metric", "MySQL", "DynamoDB", "Winner", "Margin"])

# ---------- Per-Operation Breakdown ----------
ops = []
for op in ["create_cart", "add_items", "get_cart"]:
    mysql_avg = df[(df["database"] == "MySQL") & (df["operation"] == op)]["response_time"].mean()
    dynamo_avg = df[(df["database"] == "DynamoDB") & (df["operation"] == op)]["response_time"].mean()
    ops.append({
        "Operation": op.upper(),
        "MySQL Avg (ms)": mysql_avg,
        "DynamoDB Avg (ms)": dynamo_avg,
        "Faster By": abs(mysql_avg - dynamo_avg)
    })
op_df = pd.DataFrame(ops)

# ---------- Charts ----------
# 1) Overall average response time
plt.figure(figsize=(8, 5))
plt.bar(["MySQL", "DynamoDB"], [mysql_series.mean(), dynamo_series.mean()])
plt.title("Overall Average Response Time (ms)")
plt.ylabel("Response Time (ms)")
plt.tight_layout()
plt.savefig("chart_overall_avg.png")
plt.close()

# 2) Average response time per operation
plt.figure(figsize=(9, 5))
labels, vals = [], []
for db in ["MySQL", "DynamoDB"]:
    for op in ["create_cart", "add_items", "get_cart"]:
        labels.append(f"{op} ({db})")
        vals.append(df[(df["database"] == db) & (df["operation"] == op)]["response_time"].mean())
plt.bar(labels, vals)
plt.xticks(rotation=35, ha="right")
plt.ylabel("Response Time (ms)")
plt.title("Average Response Time by Operation")
plt.tight_layout()
plt.savefig("chart_avg_by_operation.png")
plt.close()

# ---------- Generate Markdown Report ----------
report = f"""# STEP III: Database Comparison & Analysis

## Part 0: Verification
MySQL 150 records, DynamoDB 150 records ✅  
Data file: `{COMBINED_FILE}`

## Part 1: Performance Comparison

| Metric | MySQL | DynamoDB | Winner | Margin |
|--|--:|--:|--:|--:|
"""
for _, r in overall.iterrows():
    report += f"| {r.Metric} | {fmt(r.MySQL)} | {fmt(r.DynamoDB)} | {r.Winner} | {fmt(r.Margin)} |\n"

report += "\n### Operation Breakdown\n\n| Operation | MySQL Avg (ms) | DynamoDB Avg (ms) | Faster By |\n|--|--:|--:|--:|\n"
for _, r in op_df.iterrows():
    report += f"| {r.Operation} | {fmt(r['MySQL Avg (ms)'])} | {fmt(r['DynamoDB Avg (ms)'])} | {fmt(r['Faster By'])} |\n"


with open(REPORT_FILE, "w", encoding="utf-8") as f:
    f.write(report)
print(f"✅ Report generated: {REPORT_FILE}")
