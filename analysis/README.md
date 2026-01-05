# Deployment Analysis Report - LocalStack vs AWS

This directory contains the comprehensive analysis report comparing LocalStack and AWS deployments for a shopping cart REST API.

## 📄 Main Report

**[DEPLOYMENT_ANALYSIS_REPORT.md](../DEPLOYMENT_ANALYSIS_REPORT.md)** - Complete 5-page analysis report

## 📊 Performance Charts

All visualizations are located in the `charts/` directory:

1. **baseline_performance_comparison.png** - Response time metrics comparison
2. **load_scaling_comparison.png** - Throughput under increasing load
3. **cold_start_comparison.png** - Infrastructure startup time analysis
4. **cost_comparison.png** - Economic comparison of both environments
5. **latency_heatmap.png** - Operation-specific latency breakdown

## 🗂️ Test Data

### DynamoDB Test Results

- `../dynamodb/tests/aws_baseline_results.json`
- `../dynamodb/tests/aws_load_scaling_results.json`
- `../dynamodb/tests/aws_failure_*.json`
- `../dynamodb/tests/cold_start_aws_results.json`
- `../dynamodb/tests/localstack_*.json`

### MySQL Test Results

- `../mysql/tests/aws_baseline_results.json`
- `../mysql/tests/aws_load_scaling_results.json`
- `../mysql/tests/aws_consistency_*.json`
- `../mysql/tests/cold_start_aws_results.json`
- `../mysql/tests/localstack_*.json`

## 🔧 Regenerating Charts

To regenerate the performance charts:

```bash
cd analysis
python3 generate_charts.py
```

Requirements:

- Python 3.7+
- matplotlib
- numpy
- pandas

Install dependencies:

```bash
pip3 install matplotlib numpy pandas
```

## 📈 Key Findings Summary

### LocalStack Advantages

- ✅ **5-20× faster** iteration cycles
- ✅ **95%+ cost savings** (zero infrastructure costs)
- ✅ **Near-instant** cold starts (< 11ms)
- ✅ Ideal for development and testing

### AWS Advantages

- ✅ **Production-grade reliability** (99.9%+ uptime)
- ✅ **Superior tail latency** (P99: 78ms vs 518ms)
- ✅ **Predictable scaling** under load
- ✅ **Advanced features** (IAM, CloudWatch, multi-region)

### Recommendation

Use a **hybrid approach**:

- 90% development time on LocalStack
- 10% validation time on AWS staging
- 100% production traffic on AWS

**Result:** 85% cost reduction while maintaining production quality

## 🎯 Use Cases

| Scenario                   | Recommended Environment |
| -------------------------- | ----------------------- |
| Local development          | LocalStack              |
| Unit testing               | LocalStack              |
| CI/CD pre-deployment       | LocalStack              |
| Integration testing        | LocalStack              |
| Performance validation     | AWS                     |
| Load testing (> 500 users) | AWS                     |
| Production deployment      | AWS                     |
| Learning AWS services      | LocalStack              |

## 📚 Repository Structure

```
final/
├── DEPLOYMENT_ANALYSIS_REPORT.md  ← Main report
├── analysis/
│   ├── README.md                   ← This file
│   ├── generate_charts.py          ← Chart generation script
│   ├── analyze_results.py          ← Data analysis script
│   ├── combined_results.json       ← Aggregated test data
│   ├── step3_report.md             ← Preliminary analysis
│   └── charts/                     ← Generated visualizations
├── dynamodb/
│   ├── shopping-cart-api/          ← DynamoDB Go implementation
│   ├── tests/                      ← DynamoDB test results
│   └── terraform/                  ← AWS infrastructure code
└── mysql/
    ├── src/                        ← MySQL Node.js implementation
    ├── tests/                      ← MySQL test results
    └── terraform/                  ← AWS infrastructure code
```

## 🔗 Links

- **GitHub Repository:** [github.com/SujieZong/Distributed-Key-Value-Store](https://github.com/SujieZong/Distributed-Key-Value-Store)
- **LocalStack Documentation:** [docs.localstack.cloud](https://docs.localstack.cloud/)
- **AWS Well-Architected:** [aws.amazon.com/architecture/well-architected](https://aws.amazon.com/architecture/well-architected/)

## 📝 Citation

If you reference this analysis, please cite:

```
Zong, S. (2025). LocalStack vs AWS Deployment Analysis: A Comparative Study
of Development and Production Environments. Shopping Cart REST API Project.
```

---

**Last Updated:** November 30, 2025  
**Author:** Sujie Zong
