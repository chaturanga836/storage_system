#!/usr/bin/env python3
"""
Quick Health Check Script for Deployed Services
"""

import requests
import sys
import time
from typing import Dict, List

class HealthChecker:
    def __init__(self, ec2_ip: str = "localhost"):
        self.ec2_ip = ec2_ip
        self.services = {
            'auth-gateway': 8080,
            'tenant-node': 8001,
            'operation-node': 8086,
            'cbo-engine': 8088,
            'metadata-catalog': 8087,
            'monitoring': 8089,
            'grafana': 3000,
            'prometheus': 9090
        }
        
    def check_service_health(self, service_name: str, port: int) -> Dict:
        """Check health of a single service"""
        url = f"http://{self.ec2_ip}:{port}/health"
        
        try:
            start_time = time.time()
            response = requests.get(url, timeout=10)
            response_time = (time.time() - start_time) * 1000  # Convert to ms
            
            return {
                'service': service_name,
                'status': 'UP' if response.status_code == 200 else 'DOWN',
                'status_code': response.status_code,
                'response_time_ms': round(response_time, 2),
                'url': url,
                'error': None
            }
        except requests.exceptions.RequestException as e:
            return {
                'service': service_name,
                'status': 'DOWN',
                'status_code': None,
                'response_time_ms': None,
                'url': url,
                'error': str(e)
            }
    
    def check_all_services(self) -> List[Dict]:
        """Check health of all services"""
        print(f"🔍 Health checking services on {self.ec2_ip}...")
        print("=" * 80)
        
        results = []
        for service_name, port in self.services.items():
            result = self.check_service_health(service_name, port)
            results.append(result)
            
            # Print result immediately
            status_emoji = "✅" if result['status'] == 'UP' else "❌"
            print(f"{status_emoji} {service_name:20} | {result['status']:4} | {result['url']:35} | {result['response_time_ms'] or 'N/A':>6}ms")
            
            if result['error']:
                print(f"   Error: {result['error']}")
        
        return results
    
    def print_summary(self, results: List[Dict]):
        """Print test summary"""
        total_services = len(results)
        up_services = len([r for r in results if r['status'] == 'UP'])
        down_services = total_services - up_services
        
        avg_response_time = None
        response_times = [r['response_time_ms'] for r in results if r['response_time_ms'] is not None]
        if response_times:
            avg_response_time = round(sum(response_times) / len(response_times), 2)
        
        print("\n" + "=" * 80)
        print("📊 HEALTH CHECK SUMMARY")
        print("=" * 80)
        print(f"Total Services:     {total_services}")
        print(f"Services UP:        {up_services} ✅")
        print(f"Services DOWN:      {down_services} ❌")
        print(f"Success Rate:       {(up_services/total_services)*100:.1f}%")
        if avg_response_time:
            print(f"Avg Response Time:  {avg_response_time}ms")
        
        if up_services == total_services:
            print("\n🎉 ALL SERVICES ARE HEALTHY! 🎉")
            return True
        else:
            print(f"\n⚠️  {down_services} SERVICE(S) ARE DOWN - NEEDS ATTENTION")
            return False

def main():
    # Get EC2 IP from command line or use localhost
    ec2_ip = sys.argv[1] if len(sys.argv) > 1 else "localhost"
    
    checker = HealthChecker(ec2_ip)
    results = checker.check_all_services()
    all_healthy = checker.print_summary(results)
    
    # Exit with appropriate code
    sys.exit(0 if all_healthy else 1)

if __name__ == "__main__":
    main()
