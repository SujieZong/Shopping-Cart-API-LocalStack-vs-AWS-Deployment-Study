"""
Flash Sale Load Test using Locust
Simulates 500 concurrent users for 1 minute
Headless mode with HTML reports and charts
Works with both LocalStack and AWS
"""

import os
import json
import time
import random
from datetime import datetime
from pathlib import Path
from locust import HttpUser, task, between, events
from locust.env import Environment
from locust.stats import stats_printer, stats_history
from locust.log import setup_logging
import gevent

# Configuration
NUM_USERS = 500
DURATION_SECONDS = 60
SPAWN_RATE = 200  # Users to spawn per second

class ShoppingCartUser(HttpUser):
    """Simulates a user during a flash sale"""
    
    # Wait time between tasks (50-200ms)
    wait_time = between(0.05, 0.2)
    
    def on_start(self):
        """Initialize user - create a cart immediately on start"""
        self.customer_id = random.randint(1, 100000)
        self.cart_id = None
        
        # Create cart immediately when user starts
        with self.client.post(
            "/shopping-carts",
            json={"customer_id": self.customer_id},
            catch_response=True,
            name="CREATE_CART"
        ) as response:
            if response.status_code == 201:
                try:
                    data = response.json()
                    self.cart_id = data.get("shopping_cart_id")
                    if self.cart_id:
                        response.success()
                    else:
                        response.failure("No shopping_cart_id in response")
                except Exception as e:
                    response.failure(f"Failed to parse response: {e}")
            else:
                response.failure(f"Got status code {response.status_code}: {response.text}")
    
    @task(7)
    def add_item(self):
        """Add items to cart - 70% of operations"""
        if not self.cart_id:
            # Skip if cart creation failed
            return
        
        # Random product and quantity
        product_id = random.randint(1000, 2000)
        quantity = random.randint(1, 5)
        
        with self.client.post(
            f"/shopping-carts/{self.cart_id}/items",
            json={"product_id": product_id, "quantity": quantity},
            catch_response=True,
            name="ADD_ITEM"
        ) as response:
            if response.status_code in [200, 204]:
                response.success()
            else:
                response.failure(f"Got status code {response.status_code}: {response.text}")
    
    @task(3)
    def get_cart(self):
        """Retrieve cart information - 30% of operations"""
        if not self.cart_id:
            return
        
        with self.client.get(
            f"/shopping-carts/{self.cart_id}",
            catch_response=True,
            name="GET_CART"
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    # Validate response structure
                    if "shopping_cart_id" in data and "items" in data:
                        response.success()
                    else:
                        response.failure("Invalid response structure")
                except Exception as e:
                    response.failure(f"Failed to parse response: {e}")
            else:
                response.failure(f"Got status code {response.status_code}: {response.text}")


def determine_environment(base_url):
    """Determine if running against LocalStack or AWS"""
    if "localhost" in base_url or "127.0.0.1" in base_url:
        return "localstack"
    return "aws"


def save_results(env, environment_name, duration, output_dir):
    """Save test results to JSON file in output directory"""
    stats = env.stats
    
    # Calculate overall metrics
    total_requests = stats.total.num_requests
    total_failures = stats.total.num_failures
    success_rate = ((total_requests - total_failures) / total_requests * 100) if total_requests > 0 else 0
    
    # Get response time percentiles
    response_times = []
    for entry in stats.entries.values():
        response_times.extend(entry.response_times.items())
    
    # Calculate percentiles
    if response_times:
        sorted_times = sorted([rt for rt, count in response_times for _ in range(count)])
        total = len(sorted_times)
        percentiles = {
            "min": sorted_times[0] if sorted_times else 0,
            "p50": sorted_times[int(total * 0.50)] if sorted_times else 0,
            "p90": sorted_times[int(total * 0.90)] if sorted_times else 0,
            "p95": sorted_times[int(total * 0.95)] if sorted_times else 0,
            "p99": sorted_times[int(total * 0.99)] if sorted_times else 0,
            "max": sorted_times[-1] if sorted_times else 0,
        }
    else:
        percentiles = {"min": 0, "p50": 0, "p90": 0, "p95": 0, "p99": 0, "max": 0}
    
    # Collect operation-specific stats
    operation_stats = {}
    for key, entry in stats.entries.items():
        if entry.num_requests > 0:
            operation_stats[entry.name] = {
                "count": entry.num_requests,
                "success_count": entry.num_requests - entry.num_failures,
                "failure_count": entry.num_failures,
                "avg_response_time_ms": entry.avg_response_time,
                "min_response_time_ms": entry.min_response_time or 0,
                "max_response_time_ms": entry.max_response_time,
                "requests_per_sec": entry.total_rps,
                "p50_ms": entry.get_response_time_percentile(0.5) or 0,
                "p95_ms": entry.get_response_time_percentile(0.95) or 0,
                "p99_ms": entry.get_response_time_percentile(0.99) or 0,
            }
    
    # Compile results
    results = {
        "test_name": "Flash Sale Load Test",
        "environment": environment_name,
        "timestamp": datetime.now().isoformat(),
        "configuration": {
            "total_users": NUM_USERS,
            "duration_seconds": duration,
            "spawn_rate": SPAWN_RATE,
        },
        "summary": {
            "total_operations": total_requests,
            "success_count": total_requests - total_failures,
            "failure_count": total_failures,
            "success_rate_percent": round(success_rate, 2),
            "duration_seconds": duration,
            "total_throughput_ops_per_sec": round(stats.total.total_rps, 2),
            "avg_response_time_ms": round(stats.total.avg_response_time, 2),
        },
        "latency_percentiles": {
            "min_ms": percentiles["min"],
            "p50_ms": percentiles["p50"],
            "p90_ms": percentiles["p90"],
            "p95_ms": percentiles["p95"],
            "p99_ms": percentiles["p99"],
            "max_ms": percentiles["max"],
        },
        "operation_breakdown": operation_stats,
    }
    
    # Save to file in output directory
    filename = output_dir / f"flash_sale_{environment_name}_results.json"
    with open(filename, 'w') as f:
        json.dump(results, f, indent=2)
    
    return filename, results


def print_summary(results):
    """Print a formatted summary of the test results"""
    print("\n" + "="*70)
    print("║" + " "*20 + "FLASH SALE TEST SUMMARY" + " "*26 + "║")
    print("="*70)
    
    print(f"\nEnvironment: {results['environment']}")
    print(f"Duration: {results['configuration']['duration_seconds']} seconds")
    print(f"Total Users: {results['configuration']['total_users']}")
    print(f"Total Operations: {results['summary']['total_operations']}")
    
    print("\n" + "-"*50)
    print("SUCCESS METRICS")
    print("-"*50)
    print(f"Success Rate:  {results['summary']['success_rate_percent']:.2f}%")
    print(f"Successful:    {results['summary']['success_count']} ops")
    print(f"Failed:        {results['summary']['failure_count']} ops")
    
    print("\n" + "-"*50)
    print("THROUGHPUT METRICS")
    print("-"*50)
    print(f"Throughput:    {results['summary']['total_throughput_ops_per_sec']:.2f} ops/s")
    print(f"Avg Latency:   {results['summary']['avg_response_time_ms']:.2f} ms")
    
    print("\n" + "-"*50)
    print("LATENCY PERCENTILES")
    print("-"*50)
    print(f"Min:           {results['latency_percentiles']['min_ms']:.2f} ms")
    print(f"P50 (Median):  {results['latency_percentiles']['p50_ms']:.2f} ms")
    print(f"P90:           {results['latency_percentiles']['p90_ms']:.2f} ms")
    print(f"P95:           {results['latency_percentiles']['p95_ms']:.2f} ms")
    print(f"P99:           {results['latency_percentiles']['p99_ms']:.2f} ms")
    print(f"Max:           {results['latency_percentiles']['max_ms']:.2f} ms")
    
    print("\n" + "-"*70)
    print("OPERATION BREAKDOWN")
    print("-"*70)
    print(f"{'Operation':<15} {'Count':>7} {'Success':>8} {'Avg ms':>9} {'P95 ms':>9} {'P99 ms':>9}")
    print("-"*70)
    
    for op_name, stats in results['operation_breakdown'].items():
        success_rate = (stats['success_count'] / stats['count'] * 100) if stats['count'] > 0 else 0
        print(f"{op_name:<15} {stats['count']:>7} {success_rate:>7.2f}% "
              f"{stats['avg_response_time_ms']:>9.2f} {stats['p95_ms']:>9.2f} {stats['p99_ms']:>9.2f}")
    
    print("="*70 + "\n")


def run_load_test():
    """Run the flash sale load test in headless mode with HTML reports"""
    
    # Get base URL from environment variable
    base_url = os.getenv("BASE_URL")
    if not base_url:
        print("ERROR: BASE_URL environment variable is not set")
        print("Usage: export BASE_URL=http://localhost:8080 && python flash_sale_locust.py")
        return 1
    
    # Determine environment
    environment_name = determine_environment(base_url)
    
    # Create output directory
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    output_dir = Path(f"flash_sale_results_{environment_name}_{timestamp}")
    output_dir.mkdir(exist_ok=True)
    
    print(f"\n{'='*70}")
    print(f"{'FLASH SALE LOAD TEST - 500 USERS':^70}")
    print(f"{'='*70}")
    print(f"\nEnvironment: {environment_name}")
    print(f"Base URL: {base_url}")
    print(f"Duration: {DURATION_SECONDS} seconds")
    print(f"Concurrent Users: {NUM_USERS}")
    print(f"Spawn Rate: {SPAWN_RATE} users/second")
    print(f"Output Directory: {output_dir}")
    print(f"\nStarting test...\n")
    
    # Setup logging to file
    log_file = output_dir / "test.log"
    setup_logging("INFO", str(log_file))
    
    # Create environment
    env = Environment(user_classes=[ShoppingCartUser])
    env.host = base_url
    
    # Enable web monitor for stats collection
    web_ui = env.create_web_ui("127.0.0.1", 8089, stats_csv_writer=None)
    
    # Start test
    runner = env.create_local_runner()
    
    # Start a greenlet that periodically saves current stats to history
    gevent.spawn(stats_history, env.runner)
    
    # Start spawning users
    start_time = time.time()
    runner.start(NUM_USERS, spawn_rate=SPAWN_RATE)
    
    # Run for specified duration
    gevent.spawn_later(DURATION_SECONDS, lambda: runner.quit())
    
    # Wait for test to complete
    runner.greenlet.join()
    
    # Calculate actual duration
    actual_duration = time.time() - start_time
    
    print(f"\n✓ Test completed in {actual_duration:.2f} seconds")
    print(f"\nGenerating reports...")
    
    # Save HTML report using Locust's built-in report generator
    html_report = output_dir / "report.html"
    stats = env.stats
    stats.serialize_stats()
    
    # Use Locust's built-in HTML report
    from locust.stats import sort_stats
    from locust import __version__ as version
    
    with open(html_report, 'w') as f:
        from locust.html import get_html_report
        f.write(get_html_report(env))
    
    # Save CSV stats
    csv_stats = output_dir / "stats.csv"
    csv_history = output_dir / "stats_history.csv"
    csv_failures = output_dir / "failures.csv"
    
    with open(csv_stats, 'w') as f:
        f.write("Type,Name,Request Count,Failure Count,Median Response Time,Average Response Time,Min Response Time,Max Response Time,Average Content Size,Requests/s,Failures/s,50%,66%,75%,80%,90%,95%,98%,99%,99.9%,99.99%,100%\n")
        # Include both individual entries and aggregate stats
        # Stats entries are keyed by (name, method) tuple
        for key, entry in sorted(stats.entries.items()):
            if entry.num_requests > 0:  # Only include entries with requests
                f.write(f'"{entry.method}","{entry.name}",{entry.num_requests},{entry.num_failures},'
                       f'{entry.median_response_time},{entry.avg_response_time},'
                       f'{entry.min_response_time or 0},{entry.max_response_time},'
                       f'{entry.avg_content_length},{entry.total_rps},{entry.total_fail_per_sec},'
                       f'{entry.get_response_time_percentile(0.5) or 0},'
                       f'{entry.get_response_time_percentile(0.66) or 0},'
                       f'{entry.get_response_time_percentile(0.75) or 0},'
                       f'{entry.get_response_time_percentile(0.80) or 0},'
                       f'{entry.get_response_time_percentile(0.90) or 0},'
                       f'{entry.get_response_time_percentile(0.95) or 0},'
                       f'{entry.get_response_time_percentile(0.98) or 0},'
                       f'{entry.get_response_time_percentile(0.99) or 0},'
                       f'{entry.get_response_time_percentile(0.999) or 0},'
                       f'{entry.get_response_time_percentile(0.9999) or 0},'
                       f'{entry.get_response_time_percentile(1.0) or 0}\n')
        # Add aggregate stats
        if stats.total.num_requests > 0:
            f.write(f'"","Aggregated",{stats.total.num_requests},{stats.total.num_failures},'
                   f'{stats.total.median_response_time},{stats.total.avg_response_time},'
                   f'{stats.total.min_response_time or 0},{stats.total.max_response_time},'
                   f'{stats.total.avg_content_length},{stats.total.total_rps},{stats.total.total_fail_per_sec},'
                   f'{stats.total.get_response_time_percentile(0.5) or 0},'
                   f'{stats.total.get_response_time_percentile(0.66) or 0},'
                   f'{stats.total.get_response_time_percentile(0.75) or 0},'
                   f'{stats.total.get_response_time_percentile(0.80) or 0},'
                   f'{stats.total.get_response_time_percentile(0.90) or 0},'
                   f'{stats.total.get_response_time_percentile(0.95) or 0},'
                   f'{stats.total.get_response_time_percentile(0.98) or 0},'
                   f'{stats.total.get_response_time_percentile(0.99) or 0},'
                   f'{stats.total.get_response_time_percentile(0.999) or 0},'
                   f'{stats.total.get_response_time_percentile(0.9999) or 0},'
                   f'{stats.total.get_response_time_percentile(1.0) or 0}\n')
    
    # Save stats history (simplified version - stats over time)
    with open(csv_history, 'w') as f:
        f.write("Timestamp,Type,Name,Requests,Failures,Avg Response Time,Min,Max,RPS\n")
        # Use current stats entries as a snapshot
        import time as time_module
        current_timestamp = int(time_module.time())
        for key, entry in sorted(stats.entries.items()):
            if entry.num_requests > 0:
                f.write(f'{current_timestamp},"{entry.method}","{entry.name}",'
                       f'{entry.num_requests},{entry.num_failures},'
                       f'{entry.avg_response_time:.2f},'
                       f'{entry.min_response_time or 0:.2f},'
                       f'{entry.max_response_time:.2f},'
                       f'{entry.total_rps:.2f}\n')
    
    # Save failures
    if stats.errors:
        with open(csv_failures, 'w') as f:
            f.write("Method,Name,Error,Occurrences\n")
            for error_entry in stats.errors:
                # error_entry is a StatsError object
                method = getattr(error_entry, 'method', 'N/A')
                name = getattr(error_entry, 'name', 'N/A')
                error_msg = getattr(error_entry, 'error', str(error_entry))
                count = stats.errors[error_entry]
                # Escape quotes in error message
                error_msg_escaped = str(error_msg).replace('"', '""')
                f.write(f'"{method}","{name}","{error_msg_escaped}",{count}\n')
    
    # Save JSON results
    filename, results = save_results(env, environment_name, actual_duration, output_dir)
    print_summary(results)
    
    # Stop web UI
    web_ui.stop()
    
    print(f"\n{'='*70}")
    print(f"{'RESULTS SAVED':^70}")
    print(f"{'='*70}")
    print(f"\n📁 Output Directory: {output_dir}")
    print(f"📊 HTML Report: {html_report}")
    print(f"📋 JSON Results: {filename}")
    print(f"📈 CSV Stats: {csv_stats}")
    print(f"📉 CSV History: {csv_history}")
    if stats.errors:
        print(f"❌ Failures: {csv_failures}")
    print(f"📝 Test Log: {log_file}")
    print(f"\n{'='*70}\n")
    
    return 0





if __name__ == "__main__":
    exit(run_load_test())
