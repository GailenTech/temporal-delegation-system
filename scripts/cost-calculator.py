#!/usr/bin/env python3
"""
Interactive Cost Calculator for Temporal Purchase Approval System on GCP
Usage: python3 scripts/cost-calculator.py
"""

import json
import math
from datetime import datetime
from typing import Dict, List, Tuple

class GCPCostCalculator:
    """Calculate GCP costs for different deployment scenarios"""
    
    def __init__(self):
        # GCP pricing (USD/month) - us-central1 region
        self.pricing = {
            # Compute Engine (GKE nodes)
            'gke': {
                'e2-small': 24.27,     # 2 vCPU, 2GB RAM
                'e2-medium': 48.55,    # 2 vCPU, 4GB RAM
                'e2-standard-2': 97.11, # 2 vCPU, 8GB RAM
                'e2-standard-4': 194.22 # 4 vCPU, 16GB RAM
            },
            
            # Cloud SQL pricing
            'cloud_sql': {
                'db-f1-micro': 15.00,      # 1 vCPU, 0.6GB RAM
                'db-g1-small': 50.00,      # 1 vCPU, 1.7GB RAM
                'db-n1-standard-1': 95.00, # 1 vCPU, 3.75GB RAM
                'db-n1-standard-2': 190.00, # 2 vCPU, 7.5GB RAM
                'db-n1-standard-4': 380.00  # 4 vCPU, 15GB RAM
            },
            
            # Cloud Run pricing (per million requests)
            'cloud_run': {
                'cpu_time': 0.000024,      # per vCPU-second
                'memory_time': 0.0000025,  # per GB-second
                'requests': 0.40,          # per million requests
                'min_instances': 8.76      # per instance always allocated
            },
            
            # Storage pricing (per GB/month)
            'storage': {
                'persistent_ssd': 0.17,
                'persistent_standard': 0.04,
                'cloud_sql_ssd': 0.17
            },
            
            # Network pricing
            'network': {
                'load_balancer': 18.00,  # per month
                'egress_internet': 0.12  # per GB
            },
            
            # Monitoring and operations
            'operations': {
                'monitoring_basic': 5.00,     # per month
                'monitoring_premium': 25.00,  # per month
                'logging': 0.50,              # per GB
                'secret_manager': 0.06        # per 10K operations
            }
        }
    
    def calculate_gke_cost(self, machine_type: str, node_count: int, 
                          preemptible: bool = False) -> float:
        """Calculate GKE cluster costs"""
        base_cost = self.pricing['gke'][machine_type] * node_count
        if preemptible:
            base_cost *= 0.2  # 80% discount for preemptible
        return base_cost
    
    def calculate_cloud_sql_cost(self, machine_type: str, storage_gb: int, 
                                ha: bool = False, backup_storage: int = 0) -> float:
        """Calculate Cloud SQL costs"""
        compute_cost = self.pricing['cloud_sql'][machine_type]
        if ha:
            compute_cost *= 2  # Double for HA
        
        storage_cost = storage_gb * self.pricing['storage']['cloud_sql_ssd']
        backup_cost = backup_storage * self.pricing['storage']['cloud_sql_ssd'] * 0.08
        
        return compute_cost + storage_cost + backup_cost
    
    def calculate_cloud_run_cost(self, requests_per_month: int, 
                                avg_cpu_time_ms: int, avg_memory_mb: int,
                                min_instances: int = 0) -> float:
        """Calculate Cloud Run costs"""
        # Request cost
        request_cost = (requests_per_month / 1_000_000) * self.pricing['cloud_run']['requests']
        
        # CPU time cost (convert ms to seconds)
        cpu_seconds = (requests_per_month * avg_cpu_time_ms) / 1000
        cpu_vcpu_seconds = cpu_seconds * 1  # Assuming 1 vCPU
        cpu_cost = cpu_vcpu_seconds * self.pricing['cloud_run']['cpu_time']
        
        # Memory time cost
        memory_gb_seconds = (cpu_seconds * avg_memory_mb) / 1024
        memory_cost = memory_gb_seconds * self.pricing['cloud_run']['memory_time']
        
        # Minimum instances cost
        min_instance_cost = min_instances * self.pricing['cloud_run']['min_instances']
        
        return request_cost + cpu_cost + memory_cost + min_instance_cost
    
    def calculate_scenario_cost(self, scenario: str) -> Dict:
        """Calculate costs for predefined scenarios"""
        
        scenarios = {
            'demo': {
                'description': 'Demo environment for demonstrations',
                'usage': {
                    'requests_per_month': 10_000,
                    'avg_cpu_time_ms': 200,
                    'avg_memory_mb': 256,
                    'storage_gb': 50,
                    'egress_gb': 10
                },
                'components': {
                    'gke_nodes': ('e2-small', 1, True),  # preemptible
                    'cloud_sql': ('db-f1-micro', 20, False, 5),
                    'monitoring': 'basic'
                }
            },
            
            'staging': {
                'description': 'Staging environment for testing',
                'usage': {
                    'requests_per_month': 50_000,
                    'avg_cpu_time_ms': 300,
                    'avg_memory_mb': 512,
                    'storage_gb': 100, 
                    'egress_gb': 25
                },
                'components': {
                    'gke_nodes': ('e2-medium', 2, False),
                    'cloud_sql': ('db-n1-standard-1', 50, False, 15),
                    'monitoring': 'basic'
                }
            },
            
            'production': {
                'description': 'Production environment',
                'usage': {
                    'requests_per_month': 200_000,
                    'avg_cpu_time_ms': 500,
                    'avg_memory_mb': 1024,
                    'storage_gb': 200,
                    'egress_gb': 100
                },
                'components': {
                    'gke_nodes': ('e2-standard-2', 3, False),
                    'cloud_sql': ('db-n1-standard-2', 100, True, 50),
                    'monitoring': 'premium'
                }
            },
            
            'enterprise': {
                'description': 'Enterprise multi-region deployment',
                'usage': {
                    'requests_per_month': 1_000_000,
                    'avg_cpu_time_ms': 750,
                    'avg_memory_mb': 2048,
                    'storage_gb': 500,
                    'egress_gb': 500
                },
                'components': {
                    'gke_nodes': ('e2-standard-4', 6, False),  # Multi-region
                    'cloud_sql': ('db-n1-standard-4', 200, True, 100),
                    'monitoring': 'premium'
                }
            }
        }
        
        if scenario not in scenarios:
            raise ValueError(f"Unknown scenario: {scenario}")
        
        config = scenarios[scenario]
        costs = {}
        
        # GKE costs
        machine_type, node_count, preemptible = config['components']['gke_nodes']
        costs['gke'] = self.calculate_gke_cost(machine_type, node_count, preemptible)
        
        # Cloud SQL costs
        sql_machine, storage, ha, backup = config['components']['cloud_sql']
        costs['cloud_sql'] = self.calculate_cloud_sql_cost(sql_machine, storage, ha, backup)
        
        # Cloud Run costs
        usage = config['usage']
        costs['cloud_run_web'] = self.calculate_cloud_run_cost(
            usage['requests_per_month'], 
            usage['avg_cpu_time_ms'], 
            usage['avg_memory_mb'],
            min_instances=1 if scenario == 'production' else 0
        )
        
        costs['cloud_run_worker'] = self.calculate_cloud_run_cost(
            usage['requests_per_month'] // 10,  # Workers get fewer direct requests
            usage['avg_cpu_time_ms'] * 2,       # But longer processing time
            usage['avg_memory_mb'] * 2,         # And more memory
            min_instances=1 if scenario in ['production', 'enterprise'] else 0
        )
        
        # Storage costs
        costs['storage'] = usage['storage_gb'] * self.pricing['storage']['persistent_ssd']
        
        # Network costs
        costs['load_balancer'] = self.pricing['network']['load_balancer']
        costs['egress'] = usage['egress_gb'] * self.pricing['network']['egress_internet']
        
        # Monitoring costs
        monitoring_type = config['components']['monitoring']
        if monitoring_type == 'basic':
            costs['monitoring'] = self.pricing['operations']['monitoring_basic']
        else:
            costs['monitoring'] = self.pricing['operations']['monitoring_premium']
        
        # Additional costs for enterprise
        if scenario == 'enterprise':
            costs['security'] = 100  # Cloud Armor, VPC, etc.
            costs['support'] = 200   # Premium support
        
        total_cost = sum(costs.values())
        
        return {
            'scenario': scenario,
            'description': config['description'],
            'monthly_cost': total_cost,
            'annual_cost': total_cost * 12,
            'cost_breakdown': costs,
            'usage_stats': usage
        }
    
    def compare_scenarios(self, scenarios: List[str]) -> Dict:
        """Compare multiple scenarios"""
        results = {}
        for scenario in scenarios:
            results[scenario] = self.calculate_scenario_cost(scenario)
        return results
    
    def generate_report(self, scenarios: List[str]) -> str:
        """Generate a detailed cost report"""
        comparison = self.compare_scenarios(scenarios)
        
        report = []
        report.append("=" * 80)
        report.append("TEMPORAL PURCHASE APPROVAL SYSTEM - GCP COST ANALYSIS")
        report.append(f"Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        report.append("=" * 80)
        report.append("")
        
        # Summary table
        report.append("COST SUMMARY")
        report.append("-" * 50)
        report.append(f"{'Scenario':<15} {'Monthly':<12} {'Annual':<12} {'Description'}")
        report.append("-" * 50)
        
        for scenario, data in comparison.items():
            monthly = f"${data['monthly_cost']:.0f}"
            annual = f"${data['annual_cost']:.0f}"
            desc = data['description'][:30]
            report.append(f"{scenario:<15} {monthly:<12} {annual:<12} {desc}")
        
        report.append("")
        
        # Detailed breakdown for each scenario
        for scenario, data in comparison.items():
            report.append(f"DETAILED BREAKDOWN: {scenario.upper()}")
            report.append("-" * 40)
            report.append(f"Description: {data['description']}")
            report.append(f"Monthly Cost: ${data['monthly_cost']:.2f}")
            report.append(f"Annual Cost: ${data['annual_cost']:.2f}")
            report.append("")
            
            report.append("Cost Components:")
            for component, cost in data['cost_breakdown'].items():
                percentage = (cost / data['monthly_cost']) * 100
                report.append(f"  {component:<20}: ${cost:>8.2f} ({percentage:>5.1f}%)")
            
            report.append("")
            report.append("Usage Statistics:")
            for stat, value in data['usage_stats'].items():
                if isinstance(value, int):
                    report.append(f"  {stat:<20}: {value:>12,}")
                else:
                    report.append(f"  {stat:<20}: {value:>12}")
            
            report.append("")
            report.append("")
        
        # Recommendations
        report.append("RECOMMENDATIONS")
        report.append("-" * 40)
        report.append("1. START WITH DEMO: Begin with demo environment ($65-85/month)")
        report.append("2. STAGING FOR TESTING: Use staging for integration testing ($150-200/month)")
        report.append("3. PRODUCTION SCALING: Scale to production when ready ($500-650/month)")
        report.append("4. COST OPTIMIZATION:")
        report.append("   - Use preemptible instances for development")
        report.append("   - Enable auto-scaling to optimize costs")
        report.append("   - Monitor usage and adjust resources accordingly")
        report.append("   - Consider committed use discounts for production")
        
        return "\n".join(report)

def main():
    """Interactive cost calculator"""
    calculator = GCPCostCalculator()
    
    print("🧮 Temporal Purchase Approval System - GCP Cost Calculator")
    print("=" * 60)
    print()
    
    while True:
        print("Options:")
        print("1. Calculate specific scenario")
        print("2. Compare all scenarios")
        print("3. Generate detailed report")
        print("4. Custom calculation")
        print("5. Exit")
        print()
        
        choice = input("Enter your choice (1-5): ")
        
        if choice == '1':
            print("\nAvailable scenarios: demo, staging, production, enterprise")
            scenario = input("Enter scenario: ").lower()
            try:
                result = calculator.calculate_scenario_cost(scenario)
                print(f"\n{scenario.upper()} Environment:")
                print(f"Monthly Cost: ${result['monthly_cost']:.2f}")
                print(f"Annual Cost: ${result['annual_cost']:.2f}")
                print("\nTop cost components:")
                sorted_costs = sorted(result['cost_breakdown'].items(), 
                                    key=lambda x: x[1], reverse=True)[:3]
                for component, cost in sorted_costs:
                    print(f"  {component}: ${cost:.2f}")
            except ValueError as e:
                print(f"Error: {e}")
        
        elif choice == '2':
            scenarios = ['demo', 'staging', 'production', 'enterprise']
            comparison = calculator.compare_scenarios(scenarios)
            
            print(f"\n{'Scenario':<12} {'Monthly':<10} {'Annual':<12}")
            print("-" * 35)
            for scenario, data in comparison.items():
                monthly = f"${data['monthly_cost']:.0f}"
                annual = f"${data['annual_cost']:.0f}"
                print(f"{scenario:<12} {monthly:<10} {annual:<12}")
        
        elif choice == '3':
            scenarios = ['demo', 'staging', 'production', 'enterprise']
            report = calculator.generate_report(scenarios)
            
            # Save to file
            filename = f"gcp-cost-report-{datetime.now().strftime('%Y%m%d-%H%M%S')}.txt"
            with open(filename, 'w') as f:
                f.write(report)
            
            print(f"\nDetailed report saved to: {filename}")
            print("\nReport preview:")
            print(report[:1000] + "..." if len(report) > 1000 else report)
        
        elif choice == '4':
            print("\n🔧 Custom Cost Calculator")
            print("Enter your usage estimates:")
            
            try:
                requests = int(input("Requests per month: "))
                cpu_time = int(input("Average CPU time per request (ms): "))
                memory = int(input("Average memory per request (MB): "))
                
                cost = calculator.calculate_cloud_run_cost(requests, cpu_time, memory)
                print(f"\nEstimated Cloud Run cost: ${cost:.2f}/month")
                
            except ValueError:
                print("Please enter valid numbers")
        
        elif choice == '5':
            break
        
        else:
            print("Invalid choice. Please try again.")
        
        print("\n" + "="*60 + "\n")

if __name__ == "__main__":
    main()