#!/usr/bin/env python3
"""
Comprehensive Results Analyzer for LocalStack vs AWS Testing
Analyzes all test results and generates comparison report
"""

import json
import glob
import os
from typing import Dict, List, Any
from pathlib import Path

class TestResultsAnalyzer:
    def __init__(self, test_dir: str = "."):
        self.test_dir = Path(test_dir)
        self.results = {
            "localstack": {},
            "aws": {}
        }
    
    def load_all_results(self):
        """Load all JSON result files"""
        print("🔍 Loading test results...\n")
        
        # Baseline results
        self._load_json("localstack_baseline_results.json", "localstack", "baseline")
        self._load_json("aws_baseline_results.json", "aws", "baseline")
        
        # Load scaling results
        self._load_json("localstack_load_scaling_results.json", "localstack", "load_scaling")
        self._load_json("aws_load_scaling_results.json", "aws", "load_scaling")
        
        # Cold start results
        self._load_json("cold_start_localstack_results.json", "localstack", "cold_start")
        self._load_json("cold_start_aws_results.json", "aws", "cold_start")
        
        # Failure mode results
        for test_type in ["invalid_requests", "missing_resources", "malformed_data"]:
            self._load_json(f"localstack_failure_{test_type}.json", "localstack", f"failure_{test_type}")
            self._load_json(f"aws_failure_{test_type}.json", "aws", f"failure_{test_type}")
    
    def _load_json(self, filename: str, env: str, test_name: str):
        """Load a single JSON file"""
        filepath = self.test_dir / filename
        if filepath.exists():
            with open(filepath, 'r') as f:
                self.results[env][test_name] = json.load(f)
            print(f"  ✓ Loaded {filename}")
        else:
            print(f"  ✗ Missing {filename}")
    
    def analyze_baseline(self) -> Dict[str, Any]:
        """Analyze baseline performance (150 operations)"""
        print("\n" + "="*70)
        print("📊 BASELINE PERFORMANCE COMPARISON (150 Operations)")
        print("="*70)
        
        comparison = {}
        
        for env in ["localstack", "aws"]:
            if "baseline" not in self.results[env]:
                print(f"  ✗ No baseline data for {env}")
                continue
            
            data = self.results[env]["baseline"]
            
            # Calculate statistics
            response_times = [r["response_time"] for r in data]
            success_count = sum(1 for r in data if r["success"])
            
            # Group by operation
            ops = {}
            for r in data:
                op = r["operation"]
                if op not in ops:
                    ops[op] = []
                ops[op].append(r["response_time"])
            
            stats = {
                "total_ops": len(data),
                "success_count": success_count,
                "success_rate": (success_count / len(data) * 100) if data else 0,
                "avg_latency": sum(response_times) / len(response_times) if response_times else 0,
                "p50": self._percentile(response_times, 50),
                "p95": self._percentile(response_times, 95),
                "p99": self._percentile(response_times, 99),
                "min": min(response_times) if response_times else 0,
                "max": max(response_times) if response_times else 0,
                "operations": {}
            }
            
            # Per-operation stats
            for op, times in ops.items():
                stats["operations"][op] = {
                    "count": len(times),
                    "avg": sum(times) / len(times) if times else 0,
                    "p95": self._percentile(times, 95)
                }
            
            comparison[env] = stats
        
        # Print comparison
        self._print_baseline_comparison(comparison)
        
        return comparison
    
    def analyze_load_scaling(self) -> Dict[str, Any]:
        """Analyze load scaling behavior"""
        print("\n" + "="*70)
        print("📈 LOAD SCALING BEHAVIOR")
        print("="*70)
        
        comparison = {}
        
        for env in ["localstack", "aws"]:
            if "load_scaling" not in self.results[env]:
                print(f"  ✗ No load scaling data for {env}")
                continue
            
            data = self.results[env]["load_scaling"]
            comparison[env] = []
            
            for result in data:
                comparison[env].append({
                    "load_level": result["load_level"],
                    "throughput": result["throughput"],
                    "avg_latency": result["avg_response_time_ms"],
                    "p95": result["p95_ms"],
                    "p99": result["p99_ms"],
                    "success_rate": result["success_rate_percent"]
                })
        
        self._print_load_scaling_comparison(comparison)
        
        return comparison
    
    def analyze_cold_start(self) -> Dict[str, Any]:
        """Analyze cold start times"""
        print("\n" + "="*70)
        print("🥶 COLD START COMPARISON")
        print("="*70)
        
        comparison = {}
        
        for env in ["localstack", "aws"]:
            if "cold_start" not in self.results[env]:
                print(f"  ✗ No cold start data for {env}")
                continue
            
            data = self.results[env]["cold_start"]
            comparison[env] = {
                "infrastructure_up_time": data["infrastructure_up_time_seconds"],
                "first_success_time": data["first_success_time_seconds"],
                "total_cold_start": data["total_cold_start_time_seconds"],
                "first_request_attempts": data["first_request_attempts"],
                "health_check_latency": data["health_check_latency_ms"],
                "first_create_latency": data["first_create_latency_ms"],
                "consecutive_successes": data["consecutive_successes"]
            }
        
        self._print_cold_start_comparison(comparison)
        
        return comparison
    
    def analyze_failure_modes(self) -> Dict[str, Any]:
        """Analyze failure mode handling"""
        print("\n" + "="*70)
        print("❌ FAILURE MODE COMPARISON")
        print("="*70)
        
        comparison = {}
        
        for env in ["localstack", "aws"]:
            comparison[env] = {}
            
            for test_type in ["invalid_requests", "missing_resources", "malformed_data"]:
                key = f"failure_{test_type}"
                if key not in self.results[env]:
                    continue
                
                data = self.results[env][key]
                total = len(data)
                correct = sum(1 for r in data if r["success"])
                
                comparison[env][test_type] = {
                    "total_tests": total,
                    "correct_handling": correct,
                    "correct_rate": (correct / total * 100) if total else 0
                }
        
        self._print_failure_mode_comparison(comparison)
        
        return comparison
    
    def _percentile(self, values: List[float], p: float) -> float:
        """Calculate percentile"""
        if not values:
            return 0
        sorted_values = sorted(values)
        index = int(len(sorted_values) * p / 100)
        return sorted_values[min(index, len(sorted_values) - 1)]
    
    def _print_baseline_comparison(self, comparison: Dict):
        """Print baseline comparison table"""
        if "localstack" not in comparison or "aws" not in comparison:
            print("  ⚠️  Incomplete data for comparison")
            return
        
        ls = comparison["localstack"]
        aws = comparison["aws"]
        
        print(f"\n{'Metric':<30} {'LocalStack':>15} {'AWS':>15} {'Difference':>15}")
        print("-" * 77)
        
        print(f"{'Total Operations':<30} {ls['total_ops']:>15} {aws['total_ops']:>15} {aws['total_ops']-ls['total_ops']:>15}")
        print(f"{'Success Rate':<30} {ls['success_rate']:>14.2f}% {aws['success_rate']:>14.2f}% {aws['success_rate']-ls['success_rate']:>14.2f}%")
        print(f"{'Average Latency (ms)':<30} {ls['avg_latency']:>15.2f} {aws['avg_latency']:>15.2f} {aws['avg_latency']-ls['avg_latency']:>15.2f}")
        print(f"{'P50 Latency (ms)':<30} {ls['p50']:>15.2f} {aws['p50']:>15.2f} {aws['p50']-ls['p50']:>15.2f}")
        print(f"{'P95 Latency (ms)':<30} {ls['p95']:>15.2f} {aws['p95']:>15.2f} {aws['p95']-ls['p95']:>15.2f}")
        print(f"{'P99 Latency (ms)':<30} {ls['p99']:>15.2f} {aws['p99']:>15.2f} {aws['p99']-ls['p99']:>15.2f}")
        print(f"{'Min Latency (ms)':<30} {ls['min']:>15.2f} {aws['min']:>15.2f} {aws['min']-ls['min']:>15.2f}")
        print(f"{'Max Latency (ms)':<30} {ls['max']:>15.2f} {aws['max']:>15.2f} {aws['max']-ls['max']:>15.2f}")
        
        print("\n📊 Per-Operation Breakdown:")
        for op in ["create_cart", "add_items", "get_cart"]:
            if op in ls["operations"] and op in aws["operations"]:
                ls_op = ls["operations"][op]
                aws_op = aws["operations"][op]
                print(f"\n  {op.upper()}:")
                print(f"    Count: LocalStack={ls_op['count']}, AWS={aws_op['count']}")
                print(f"    Avg Latency: LocalStack={ls_op['avg']:.2f}ms, AWS={aws_op['avg']:.2f}ms (diff: {aws_op['avg']-ls_op['avg']:+.2f}ms)")
                print(f"    P95 Latency: LocalStack={ls_op['p95']:.2f}ms, AWS={aws_op['p95']:.2f}ms (diff: {aws_op['p95']-ls_op['p95']:+.2f}ms)")
    
    def _print_load_scaling_comparison(self, comparison: Dict):
        """Print load scaling comparison"""
        if "localstack" not in comparison or "aws" not in comparison:
            print("  ⚠️  Incomplete data for comparison")
            return
        
        print(f"\n{'Load Level':<15} {'Environment':<15} {'Throughput':>15} {'Avg Latency':>15} {'P95':>12} {'P99':>12} {'Success %':>12}")
        print("-" * 97)
        
        # Combine and sort by load level
        all_results = []
        for env in ["localstack", "aws"]:
            for result in comparison[env]:
                all_results.append((result["load_level"], env, result))
        
        all_results.sort(key=lambda x: (x[0], x[1]))
        
        for load_level, env, result in all_results:
            print(f"{load_level:<15} {env:<15} {result['throughput']:>14.2f}x {result['avg_latency']:>14.2f}ms {result['p95']:>11.2f}ms {result['p99']:>11.2f}ms {result['success_rate']:>11.2f}%")
    
    def _print_cold_start_comparison(self, comparison: Dict):
        """Print cold start comparison"""
        if "localstack" not in comparison or "aws" not in comparison:
            print("  ⚠️  Incomplete data for comparison")
            return
        
        ls = comparison["localstack"]
        aws = comparison["aws"]
        
        print(f"\n{'Metric':<35} {'LocalStack':>15} {'AWS':>15} {'Difference':>15}")
        print("-" * 82)
        
        print(f"{'Infrastructure Up Time (s)':<35} {ls['infrastructure_up_time']:>15.2f} {aws['infrastructure_up_time']:>15.2f} {aws['infrastructure_up_time']-ls['infrastructure_up_time']:>15.2f}")
        print(f"{'First Success Time (s)':<35} {ls['first_success_time']:>15.2f} {aws['first_success_time']:>15.2f} {aws['first_success_time']-ls['first_success_time']:>15.2f}")
        print(f"{'Total Cold Start Time (s)':<35} {ls['total_cold_start']:>15.2f} {aws['total_cold_start']:>15.2f} {aws['total_cold_start']-ls['total_cold_start']:>15.2f}")
        print(f"{'First Request Attempts':<35} {ls['first_request_attempts']:>15} {aws['first_request_attempts']:>15} {aws['first_request_attempts']-ls['first_request_attempts']:>15}")
        print(f"{'Health Check Latency (ms)':<35} {ls['health_check_latency']:>15.2f} {aws['health_check_latency']:>15.2f} {aws['health_check_latency']-ls['health_check_latency']:>15.2f}")
        print(f"{'First Create Latency (ms)':<35} {ls['first_create_latency']:>15.2f} {aws['first_create_latency']:>15.2f} {aws['first_create_latency']-ls['first_create_latency']:>15.2f}")
        print(f"{'Consecutive Successes (out of 5)':<35} {ls['consecutive_successes']:>15} {aws['consecutive_successes']:>15} {aws['consecutive_successes']-ls['consecutive_successes']:>15}")
    
    def _print_failure_mode_comparison(self, comparison: Dict):
        """Print failure mode comparison"""
        if "localstack" not in comparison or "aws" not in comparison:
            print("  ⚠️  Incomplete data for comparison")
            return
        
        print(f"\n{'Test Type':<30} {'Environment':<15} {'Total Tests':>15} {'Correct':>12} {'Rate':>10}")
        print("-" * 84)
        
        for test_type in ["invalid_requests", "missing_resources", "malformed_data"]:
            for env in ["localstack", "aws"]:
                if test_type in comparison[env]:
                    data = comparison[env][test_type]
                    print(f"{test_type:<30} {env:<15} {data['total_tests']:>15} {data['correct_handling']:>12} {data['correct_rate']:>9.1f}%")
    
    def generate_summary_report(self):
        """Generate a complete summary report"""
        print("\n" + "="*70)
        print("📝 GENERATING COMPREHENSIVE SUMMARY REPORT")
        print("="*70)
        
        self.load_all_results()
        
        baseline = self.analyze_baseline()
        load_scaling = self.analyze_load_scaling()
        cold_start = self.analyze_cold_start()
        failure_modes = self.analyze_failure_modes()
        
        # Save summary to file
        summary = {
            "baseline": baseline,
            "load_scaling": load_scaling,
            "cold_start": cold_start,
            "failure_modes": failure_modes,
            "timestamp": __import__("datetime").datetime.now().isoformat()
        }
        
        output_file = self.test_dir / "comprehensive_summary.json"
        with open(output_file, 'w') as f:
            json.dump(summary, f, indent=2)
        
        print(f"\n✅ Summary report saved to: {output_file}")
        
        # Print key findings
        self._print_key_findings(baseline, load_scaling, cold_start, failure_modes)
    
    def _print_key_findings(self, baseline, load_scaling, cold_start, failure_modes):
        """Print key findings"""
        print("\n" + "="*70)
        print("🎯 KEY FINDINGS")
        print("="*70)
        
        if baseline.get("localstack") and baseline.get("aws"):
            ls_avg = baseline["localstack"]["avg_latency"]
            aws_avg = baseline["aws"]["avg_latency"]
            diff_pct = ((aws_avg - ls_avg) / ls_avg * 100) if ls_avg > 0 else 0
            
            print(f"\n1. Baseline Performance:")
            print(f"   • AWS is {diff_pct:+.1f}% {'slower' if diff_pct > 0 else 'faster'} than LocalStack on average")
            print(f"   • LocalStack avg latency: {ls_avg:.2f}ms")
            print(f"   • AWS avg latency: {aws_avg:.2f}ms")
        
        if cold_start.get("localstack") and cold_start.get("aws"):
            ls_cold = cold_start["localstack"]["total_cold_start"]
            aws_cold = cold_start["aws"]["total_cold_start"]
            
            print(f"\n2. Cold Start Times:")
            print(f"   • LocalStack cold start: {ls_cold:.2f}s")
            print(f"   • AWS cold start: {aws_cold:.2f}s")
            print(f"   • AWS is {aws_cold/ls_cold:.1f}x slower to cold start")
        
        if load_scaling.get("localstack") and load_scaling.get("aws"):
            print(f"\n3. Load Scaling:")
            print(f"   • Check if latency increases linearly with load")
            print(f"   • Compare throughput degradation between environments")
        
        print(f"\n4. Error Handling:")
        print(f"   • Both environments should handle errors identically")
        print(f"   • Review failure mode results for discrepancies")
        
        print("\n" + "="*70)

def main():
    analyzer = TestResultsAnalyzer()
    analyzer.generate_summary_report()

if __name__ == "__main__":
    main()
