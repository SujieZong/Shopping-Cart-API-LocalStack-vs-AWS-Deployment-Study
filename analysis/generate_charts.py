#!/usr/bin/env python3
"""
Performance Visualization Script
Generates comparison charts for LocalStack vs AWS deployment analysis
"""

import json
import matplotlib.pyplot as plt
import numpy as np
from pathlib import Path

# Set style
plt.style.use('seaborn-v0_8-darkgrid')
plt.rcParams['figure.figsize'] = (12, 8)
plt.rcParams['font.size'] = 10

# Data paths
BASE_DIR = Path(__file__).parent.parent  # Go up to 'final' directory
DYNAMODB_DIR = BASE_DIR / 'dynamodb' / 'tests'
MYSQL_DIR = BASE_DIR / 'mysql' / 'tests'
OUTPUT_DIR = BASE_DIR / 'analysis' / 'charts'
OUTPUT_DIR.mkdir(exist_ok=True, parents=True)


def load_json(filepath):
    """Load JSON file"""
    with open(filepath, 'r') as f:
        return json.load(f)


def calculate_percentile(data, percentile):
    """Calculate percentile from list of values"""
    return np.percentile(data, percentile)


def extract_baseline_metrics(results):
    """Extract metrics from baseline test results"""
    response_times = [r.get('response_time_ms', r.get('response_time', 0)) for r in results if r.get('success', True)]
    return {
        'avg': np.mean(response_times),
        'p50': calculate_percentile(response_times, 50),
        'p95': calculate_percentile(response_times, 95),
        'p99': calculate_percentile(response_times, 99),
        'success_rate': sum(1 for r in results if r.get('success', True)) / len(results) * 100
    }


def plot_baseline_comparison():
    """Plot 1: Baseline Performance Comparison"""
    # Load data
    mysql_local = load_json(MYSQL_DIR / 'localstack_baseline_results.json')
    mysql_aws = load_json(MYSQL_DIR / 'aws_baseline_results.json')
    dynamo_local = load_json(DYNAMODB_DIR / 'localstack_baseline_results.json')
    dynamo_aws = load_json(DYNAMODB_DIR / 'aws_baseline_results.json')
    
    # Extract metrics
    mysql_local_metrics = extract_baseline_metrics(mysql_local)
    mysql_aws_metrics = extract_baseline_metrics(mysql_aws)
    dynamo_local_metrics = extract_baseline_metrics(dynamo_local)
    dynamo_aws_metrics = extract_baseline_metrics(dynamo_aws)
    
    # Create subplots
    fig, ((ax1, ax2), (ax3, ax4)) = plt.subplots(2, 2, figsize=(16, 12))
    
    # Plot 1: Average Response Time
    categories = ['MySQL', 'DynamoDB']
    local_avgs = [mysql_local_metrics['avg'], dynamo_local_metrics['avg']]
    aws_avgs = [mysql_aws_metrics['avg'], dynamo_aws_metrics['avg']]
    
    x = np.arange(len(categories))
    width = 0.35
    
    ax1.bar(x - width/2, local_avgs, width, label='LocalStack', color='#2ecc71', alpha=0.8)
    ax1.bar(x + width/2, aws_avgs, width, label='AWS', color='#3498db', alpha=0.8)
    ax1.set_ylabel('Response Time (ms)')
    ax1.set_title('Average Response Time Comparison')
    ax1.set_xticks(x)
    ax1.set_xticklabels(categories)
    ax1.legend()
    ax1.grid(axis='y', alpha=0.3)
    
    # Plot 2: P99 Latency
    local_p99 = [mysql_local_metrics['p99'], dynamo_local_metrics['p99']]
    aws_p99 = [mysql_aws_metrics['p99'], dynamo_aws_metrics['p99']]
    
    ax2.bar(x - width/2, local_p99, width, label='LocalStack', color='#2ecc71', alpha=0.8)
    ax2.bar(x + width/2, aws_p99, width, label='AWS', color='#3498db', alpha=0.8)
    ax2.set_ylabel('Response Time (ms)')
    ax2.set_title('P99 Latency Comparison')
    ax2.set_xticks(x)
    ax2.set_xticklabels(categories)
    ax2.legend()
    ax2.grid(axis='y', alpha=0.3)
    
    # Plot 3: Latency Distribution (MySQL)
    metrics_labels = ['Avg', 'P50', 'P95', 'P99']
    mysql_local_dist = [mysql_local_metrics['avg'], mysql_local_metrics['p50'], 
                        mysql_local_metrics['p95'], mysql_local_metrics['p99']]
    mysql_aws_dist = [mysql_aws_metrics['avg'], mysql_aws_metrics['p50'],
                      mysql_aws_metrics['p95'], mysql_aws_metrics['p99']]
    
    x_dist = np.arange(len(metrics_labels))
    ax3.plot(x_dist, mysql_local_dist, marker='o', linewidth=2, markersize=8, 
             label='LocalStack', color='#2ecc71')
    ax3.plot(x_dist, mysql_aws_dist, marker='s', linewidth=2, markersize=8,
             label='AWS', color='#3498db')
    ax3.set_ylabel('Response Time (ms)')
    ax3.set_title('MySQL Latency Distribution')
    ax3.set_xticks(x_dist)
    ax3.set_xticklabels(metrics_labels)
    ax3.legend()
    ax3.grid(True, alpha=0.3)
    
    # Plot 4: Latency Distribution (DynamoDB)
    dynamo_local_dist = [dynamo_local_metrics['avg'], dynamo_local_metrics['p50'],
                         dynamo_local_metrics['p95'], dynamo_local_metrics['p99']]
    dynamo_aws_dist = [dynamo_aws_metrics['avg'], dynamo_aws_metrics['p50'],
                       dynamo_aws_metrics['p95'], dynamo_aws_metrics['p99']]
    
    ax4.plot(x_dist, dynamo_local_dist, marker='o', linewidth=2, markersize=8,
             label='LocalStack', color='#2ecc71')
    ax4.plot(x_dist, dynamo_aws_dist, marker='s', linewidth=2, markersize=8,
             label='AWS', color='#3498db')
    ax4.set_ylabel('Response Time (ms)')
    ax4.set_title('DynamoDB Latency Distribution')
    ax4.set_xticks(x_dist)
    ax4.set_xticklabels(metrics_labels)
    ax4.legend()
    ax4.grid(True, alpha=0.3)
    
    plt.tight_layout()
    plt.savefig(OUTPUT_DIR / 'baseline_performance_comparison.png', dpi=300, bbox_inches='tight')
    print(f"✅ Saved: baseline_performance_comparison.png")
    plt.close()


def plot_load_scaling():
    """Plot 2: Load Scaling Performance"""
    # Load data
    mysql_local_load = load_json(MYSQL_DIR / 'localstack_load_scaling_results.json')
    mysql_aws_load = load_json(MYSQL_DIR / 'aws_load_scaling_results.json')
    dynamo_local_load = load_json(DYNAMODB_DIR / 'localstack_load_scaling_results.json')
    dynamo_aws_load = load_json(DYNAMODB_DIR / 'aws_load_scaling_results.json')
    
    # Create subplots
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(16, 6))
    
    # Plot 1: MySQL Throughput
    load_levels = [50, 100, 167]
    mysql_local_throughput = [r['throughput_ops_per_sec'] for r in mysql_local_load]
    mysql_aws_throughput = [r['throughput_ops_per_sec'] for r in mysql_aws_load]
    
    ax1.plot(load_levels, mysql_local_throughput, marker='o', linewidth=2.5, markersize=10,
             label='LocalStack', color='#2ecc71')
    ax1.plot(load_levels, mysql_aws_throughput, marker='s', linewidth=2.5, markersize=10,
             label='AWS', color='#3498db')
    ax1.set_xlabel('Concurrent Users')
    ax1.set_ylabel('Throughput (ops/sec)')
    ax1.set_title('MySQL Throughput Scaling')
    ax1.legend()
    ax1.grid(True, alpha=0.3)
    ax1.set_xticks(load_levels)
    
    # Plot 2: DynamoDB Throughput
    dynamo_local_throughput = [r['throughput_ops_per_sec'] for r in dynamo_local_load]
    dynamo_aws_throughput = [r['throughput_ops_per_sec'] for r in dynamo_aws_load]
    
    ax2.plot(load_levels, dynamo_local_throughput, marker='o', linewidth=2.5, markersize=10,
             label='LocalStack', color='#2ecc71')
    ax2.plot(load_levels, dynamo_aws_throughput, marker='s', linewidth=2.5, markersize=10,
             label='AWS', color='#3498db')
    ax2.set_xlabel('Concurrent Users')
    ax2.set_ylabel('Throughput (ops/sec)')
    ax2.set_title('DynamoDB Throughput Scaling')
    ax2.legend()
    ax2.grid(True, alpha=0.3)
    ax2.set_xticks(load_levels)
    
    plt.tight_layout()
    plt.savefig(OUTPUT_DIR / 'load_scaling_comparison.png', dpi=300, bbox_inches='tight')
    print(f"✅ Saved: load_scaling_comparison.png")
    plt.close()


def plot_cold_start():
    """Plot 3: Cold Start Time Comparison"""
    # Load data
    mysql_local_cold = load_json(MYSQL_DIR / 'cold_start_localstack_results.json')
    mysql_aws_cold = load_json(MYSQL_DIR / 'cold_start_aws_results.json')
    dynamo_local_cold = load_json(DYNAMODB_DIR / 'cold_start_localstack_results.json')
    dynamo_aws_cold = load_json(DYNAMODB_DIR / 'cold_start_aws_results.json')
    
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(16, 6))
    
    # Plot 1: Total Cold Start Time
    categories = ['MySQL', 'DynamoDB']
    local_times = [mysql_local_cold['total_cold_start_time_seconds'] * 1000,
                   dynamo_local_cold['total_cold_start_time_seconds'] * 1000]
    aws_times = [mysql_aws_cold['total_cold_start_time_seconds'] * 1000,
                 dynamo_aws_cold['total_cold_start_time_seconds'] * 1000]
    
    x = np.arange(len(categories))
    width = 0.35
    
    bars1 = ax1.bar(x - width/2, local_times, width, label='LocalStack', color='#2ecc71', alpha=0.8)
    bars2 = ax1.bar(x + width/2, aws_times, width, label='AWS', color='#3498db', alpha=0.8)
    
    ax1.set_ylabel('Cold Start Time (ms)')
    ax1.set_title('Total Cold Start Time Comparison')
    ax1.set_xticks(x)
    ax1.set_xticklabels(categories)
    ax1.legend()
    ax1.grid(axis='y', alpha=0.3)
    ax1.set_yscale('log')  # Log scale due to large difference
    
    # Add value labels
    for bars in [bars1, bars2]:
        for bar in bars:
            height = bar.get_height()
            ax1.annotate(f'{height:.1f}',
                        xy=(bar.get_x() + bar.get_width() / 2, height),
                        xytext=(0, 3),
                        textcoords="offset points",
                        ha='center', va='bottom', fontsize=9)
    
    # Plot 2: First Request Latency
    local_first = [mysql_local_cold['first_create_latency_ms'],
                   dynamo_local_cold['first_create_latency_ms']]
    aws_first = [mysql_aws_cold['first_create_latency_ms'],
                 dynamo_aws_cold['first_create_latency_ms']]
    
    bars3 = ax2.bar(x - width/2, local_first, width, label='LocalStack', color='#2ecc71', alpha=0.8)
    bars4 = ax2.bar(x + width/2, aws_first, width, label='AWS', color='#3498db', alpha=0.8)
    
    ax2.set_ylabel('First Request Latency (ms)')
    ax2.set_title('First Request Latency After Cold Start')
    ax2.set_xticks(x)
    ax2.set_xticklabels(categories)
    ax2.legend()
    ax2.grid(axis='y', alpha=0.3)
    
    # Add value labels
    for bars in [bars3, bars4]:
        for bar in bars:
            height = bar.get_height()
            ax2.annotate(f'{height:.1f}',
                        xy=(bar.get_x() + bar.get_width() / 2, height),
                        xytext=(0, 3),
                        textcoords="offset points",
                        ha='center', va='bottom', fontsize=9)
    
    plt.tight_layout()
    plt.savefig(OUTPUT_DIR / 'cold_start_comparison.png', dpi=300, bbox_inches='tight')
    print(f"✅ Saved: cold_start_comparison.png")
    plt.close()


def plot_cost_comparison():
    """Plot 4: Cost Comparison"""
    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(16, 6))
    
    # Plot 1: Monthly Infrastructure Cost
    categories = ['LocalStack', 'AWS']
    costs = [0, 50]  # USD per month
    colors = ['#2ecc71', '#3498db']
    
    bars = ax1.bar(categories, costs, color=colors, alpha=0.8, width=0.6)
    ax1.set_ylabel('Cost (USD/month)')
    ax1.set_title('Monthly Infrastructure Cost')
    ax1.grid(axis='y', alpha=0.3)
    
    # Add value labels
    for bar in bars:
        height = bar.get_height()
        ax1.annotate(f'${height:.0f}',
                    xy=(bar.get_x() + bar.get_width() / 2, height),
                    xytext=(0, 3),
                    textcoords="offset points",
                    ha='center', va='bottom', fontsize=12, fontweight='bold')
    
    # Plot 2: Developer Time Cost (per 100 test iterations)
    test_times = [8, 50]  # minutes per 100 iterations
    dev_costs = [6.67, 41.67]  # USD at $50/hour developer rate
    
    x = np.arange(len(categories))
    width = 0.35
    
    ax2_time = ax2.twinx()
    
    bars1 = ax2.bar(x - width/2, test_times, width, label='Time (min)', color='#e74c3c', alpha=0.8)
    bars2 = ax2_time.bar(x + width/2, dev_costs, width, label='Cost ($)', color='#f39c12', alpha=0.8)
    
    ax2.set_ylabel('Time (minutes)', color='#e74c3c')
    ax2_time.set_ylabel('Cost (USD)', color='#f39c12')
    ax2.set_title('Developer Time Cost (100 Test Iterations)')
    ax2.set_xticks(x)
    ax2.set_xticklabels(categories)
    ax2.tick_params(axis='y', labelcolor='#e74c3c')
    ax2_time.tick_params(axis='y', labelcolor='#f39c12')
    
    # Add legends
    ax2.legend(loc='upper left')
    ax2_time.legend(loc='upper right')
    
    ax2.grid(axis='y', alpha=0.3)
    
    plt.tight_layout()
    plt.savefig(OUTPUT_DIR / 'cost_comparison.png', dpi=300, bbox_inches='tight')
    print(f"✅ Saved: cost_comparison.png")
    plt.close()


def plot_latency_heatmap():
    """Plot 5: Latency Heatmap"""
    # Load baseline data
    mysql_local = load_json(MYSQL_DIR / 'localstack_baseline_results.json')
    mysql_aws = load_json(MYSQL_DIR / 'aws_baseline_results.json')
    dynamo_local = load_json(DYNAMODB_DIR / 'localstack_baseline_results.json')
    dynamo_aws = load_json(DYNAMODB_DIR / 'aws_baseline_results.json')
    
    # Extract by operation
    def get_op_metrics(data, operation):
        op_data = [r.get('response_time_ms', r.get('response_time', 0)) for r in data if r.get('operation') == operation]
        if not op_data:
            return [0, 0, 0, 0]
        return [
            np.mean(op_data),
            calculate_percentile(op_data, 50),
            calculate_percentile(op_data, 95),
            calculate_percentile(op_data, 99)
        ]
    
    operations = ['create_cart', 'add_items', 'get_cart']
    metrics = ['Avg', 'P50', 'P95', 'P99']
    
    # Create data matrices
    mysql_local_matrix = np.array([get_op_metrics(mysql_local, op) for op in operations])
    mysql_aws_matrix = np.array([get_op_metrics(mysql_aws, op) for op in operations])
    dynamo_local_matrix = np.array([get_op_metrics(dynamo_local, op) for op in operations])
    dynamo_aws_matrix = np.array([get_op_metrics(dynamo_aws, op) for op in operations])
    
    # Create subplots
    fig, ((ax1, ax2), (ax3, ax4)) = plt.subplots(2, 2, figsize=(16, 12))
    
    # Plot heatmaps
    im1 = ax1.imshow(mysql_local_matrix, cmap='Greens', aspect='auto')
    ax1.set_title('MySQL LocalStack Latency (ms)')
    ax1.set_xticks(np.arange(len(metrics)))
    ax1.set_yticks(np.arange(len(operations)))
    ax1.set_xticklabels(metrics)
    ax1.set_yticklabels(['Create Cart', 'Add Items', 'Get Cart'])
    plt.colorbar(im1, ax=ax1)
    
    # Add text annotations
    for i in range(len(operations)):
        for j in range(len(metrics)):
            text = ax1.text(j, i, f'{mysql_local_matrix[i, j]:.1f}',
                           ha="center", va="center", color="black", fontsize=10)
    
    im2 = ax2.imshow(mysql_aws_matrix, cmap='Blues', aspect='auto')
    ax2.set_title('MySQL AWS Latency (ms)')
    ax2.set_xticks(np.arange(len(metrics)))
    ax2.set_yticks(np.arange(len(operations)))
    ax2.set_xticklabels(metrics)
    ax2.set_yticklabels(['Create Cart', 'Add Items', 'Get Cart'])
    plt.colorbar(im2, ax=ax2)
    
    for i in range(len(operations)):
        for j in range(len(metrics)):
            text = ax2.text(j, i, f'{mysql_aws_matrix[i, j]:.1f}',
                           ha="center", va="center", color="white", fontsize=10)
    
    im3 = ax3.imshow(dynamo_local_matrix, cmap='Greens', aspect='auto')
    ax3.set_title('DynamoDB LocalStack Latency (ms)')
    ax3.set_xticks(np.arange(len(metrics)))
    ax3.set_yticks(np.arange(len(operations)))
    ax3.set_xticklabels(metrics)
    ax3.set_yticklabels(['Create Cart', 'Add Items', 'Get Cart'])
    plt.colorbar(im3, ax=ax3)
    
    for i in range(len(operations)):
        for j in range(len(metrics)):
            text = ax3.text(j, i, f'{dynamo_local_matrix[i, j]:.1f}',
                           ha="center", va="center", color="black", fontsize=10)
    
    im4 = ax4.imshow(dynamo_aws_matrix, cmap='Blues', aspect='auto')
    ax4.set_title('DynamoDB AWS Latency (ms)')
    ax4.set_xticks(np.arange(len(metrics)))
    ax4.set_yticks(np.arange(len(operations)))
    ax4.set_xticklabels(metrics)
    ax4.set_yticklabels(['Create Cart', 'Add Items', 'Get Cart'])
    plt.colorbar(im4, ax=ax4)
    
    for i in range(len(operations)):
        for j in range(len(metrics)):
            text = ax4.text(j, i, f'{dynamo_aws_matrix[i, j]:.1f}',
                           ha="center", va="center", color="white", fontsize=10)
    
    plt.tight_layout()
    plt.savefig(OUTPUT_DIR / 'latency_heatmap.png', dpi=300, bbox_inches='tight')
    print(f"✅ Saved: latency_heatmap.png")
    plt.close()


def main():
    """Generate all visualizations"""
    print("🎨 Generating performance comparison charts...")
    print("-" * 60)
    
    try:
        plot_baseline_comparison()
        plot_load_scaling()
        plot_cold_start()
        plot_cost_comparison()
        plot_latency_heatmap()
        
        print("-" * 60)
        print(f"✅ All charts generated successfully!")
        print(f"📂 Output directory: {OUTPUT_DIR}")
        print("\n📊 Generated files:")
        print("  1. baseline_performance_comparison.png")
        print("  2. load_scaling_comparison.png")
        print("  3. cold_start_comparison.png")
        print("  4. cost_comparison.png")
        print("  5. latency_heatmap.png")
        
    except Exception as e:
        print(f"❌ Error generating charts: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    main()
