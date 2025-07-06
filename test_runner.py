#!/usr/bin/env python3
"""
Comprehensive Test Runner for Storage System
"""

import os
import sys
import subprocess
import time
import json
from datetime import datetime

class TestRunner:
    def __init__(self, ec2_ip: str = "localhost"):
        self.ec2_ip = ec2_ip
        self.test_results = {
            'timestamp': datetime.now().isoformat(),
            'ec2_ip': ec2_ip,
            'tests': {}
        }
    
    def run_command(self, command: str, description: str) -> dict:
        """Run a command and capture results"""
        print(f"\n🔄 {description}")
        print("-" * 60)
        
        start_time = time.time()
        try:
            result = subprocess.run(
                command, 
                shell=True, 
                capture_output=True, 
                text=True, 
                timeout=300  # 5 minute timeout
            )
            execution_time = time.time() - start_time
            
            success = result.returncode == 0
            
            print(f"✅ {description} - {'PASSED' if success else 'FAILED'} ({execution_time:.2f}s)")
            if not success:
                print(f"Error: {result.stderr}")
            
            return {
                'success': success,
                'execution_time': execution_time,
                'stdout': result.stdout,
                'stderr': result.stderr,
                'return_code': result.returncode
            }
            
        except subprocess.TimeoutExpired:
            print(f"❌ {description} - TIMEOUT (>300s)")
            return {
                'success': False,
                'execution_time': 300,
                'stdout': '',
                'stderr': 'Test timed out after 300 seconds',
                'return_code': -1
            }
        except Exception as e:
            print(f"❌ {description} - ERROR: {e}")
            return {
                'success': False,
                'execution_time': 0,
                'stdout': '',
                'stderr': str(e),
                'return_code': -1
            }
    
    def test_health_checks(self) -> bool:
        """Phase 1: Health Checks"""
        print("\n" + "="*80)
        print("🏥 PHASE 1: HEALTH CHECKS")
        print("="*80)
        
        result = self.run_command(
            f"python health_check.py {self.ec2_ip}",
            "Service Health Verification"
        )
        
        self.test_results['tests']['health_checks'] = result
        return result['success']
    
    def test_basic_endpoints(self) -> bool:
        """Phase 2: Basic Endpoint Tests"""
        print("\n" + "="*80)
        print("🌐 PHASE 2: BASIC ENDPOINT TESTS")
        print("="*80)
        
        endpoints = [
            ('auth-gateway', 8080, '/health'),
            ('tenant-node', 8001, '/health'),
            ('metadata-catalog', 8087, '/health'),
            ('cbo-engine', 8088, '/health'),
            ('operation-node', 8086, '/health'),
            ('monitoring', 8089, '/health')
        ]
        
        all_passed = True
        endpoint_results = {}
        
        for service, port, path in endpoints:
            url = f"http://{self.ec2_ip}:{port}{path}"
            result = self.run_command(
                f"curl -s -o /dev/null -w '%{{http_code}}' {url}",
                f"Testing {service} endpoint"
            )
            
            # Check if we got 200 status code
            if result['success'] and result['stdout'].strip() == '200':
                endpoint_results[service] = {'success': True, 'status_code': 200}
            else:
                endpoint_results[service] = {'success': False, 'status_code': result['stdout'].strip()}
                all_passed = False
        
        self.test_results['tests']['basic_endpoints'] = {
            'success': all_passed,
            'endpoints': endpoint_results
        }
        
        return all_passed
    
    def test_integration_suite(self) -> bool:
        """Phase 3: Integration Tests"""
        print("\n" + "="*80)
        print("🔗 PHASE 3: INTEGRATION TESTS")
        print("="*80)
        
        # Install required dependencies first
        deps_result = self.run_command(
            "pip install aiohttp pytest requests",
            "Installing test dependencies"
        )
        
        if not deps_result['success']:
            print("❌ Failed to install test dependencies")
            self.test_results['tests']['integration'] = deps_result
            return False
        
        # Run integration tests
        result = self.run_command(
            f"python integration_test_suite.py {self.ec2_ip}",
            "Integration Test Suite"
        )
        
        self.test_results['tests']['integration'] = result
        return result['success']
    
    def test_load_testing(self) -> bool:
        """Phase 4: Basic Load Testing"""
        print("\n" + "="*80)
        print("⚡ PHASE 4: LOAD TESTING")
        print("="*80)
        
        # Simple concurrent requests test
        load_tests = [
            (f"http://{self.ec2_ip}:8001/health", "tenant-node load test"),
            (f"http://{self.ec2_ip}:8080/health", "auth-gateway load test")
        ]
        
        all_passed = True
        load_results = {}
        
        for url, description in load_tests:
            # Use curl to simulate concurrent requests
            result = self.run_command(
                f"for i in {{1..20}}; do curl -s {url} > /dev/null & done; wait",
                description
            )
            
            load_results[description] = result
            if not result['success']:
                all_passed = False
        
        self.test_results['tests']['load_testing'] = {
            'success': all_passed,
            'tests': load_results
        }
        
        return all_passed
    
    def generate_report(self):
        """Generate final test report"""
        print("\n" + "="*80)
        print("📊 COMPREHENSIVE TEST REPORT")
        print("="*80)
        
        total_tests = 0
        passed_tests = 0
        
        for test_name, test_result in self.test_results['tests'].items():
            total_tests += 1
            status = "✅ PASSED" if test_result.get('success', False) else "❌ FAILED"
            execution_time = test_result.get('execution_time', 0)
            
            print(f"{test_name:20} | {status:10} | {execution_time:6.2f}s")
            
            if test_result.get('success', False):
                passed_tests += 1
        
        success_rate = (passed_tests / total_tests) * 100 if total_tests > 0 else 0
        
        print("-" * 80)
        print(f"Total Tests:     {total_tests}")
        print(f"Passed:          {passed_tests}")
        print(f"Failed:          {total_tests - passed_tests}")
        print(f"Success Rate:    {success_rate:.1f}%")
        print(f"EC2 Instance:    {self.ec2_ip}")
        print(f"Test Time:       {self.test_results['timestamp']}")
        
        # Save detailed report
        report_file = f"test_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        with open(report_file, 'w') as f:
            json.dump(self.test_results, f, indent=2)
        
        print(f"\n📄 Detailed report saved to: {report_file}")
        
        if success_rate == 100:
            print("\n🎉 ALL TESTS PASSED! Your storage system is fully functional! 🎉")
            return True
        else:
            print(f"\n⚠️  {total_tests - passed_tests} test(s) failed. System needs attention.")
            return False

def main():
    print("🧪 COMPREHENSIVE STORAGE SYSTEM TESTING")
    print("="*80)
    
    # Get EC2 IP from command line or use localhost
    ec2_ip = sys.argv[1] if len(sys.argv) > 1 else "localhost"
    
    if ec2_ip == "localhost":
        print("⚠️  Using localhost - make sure to provide EC2 IP for remote testing")
        print("   Usage: python test_runner.py <EC2_IP_ADDRESS>")
    
    print(f"🎯 Testing target: {ec2_ip}")
    
    runner = TestRunner(ec2_ip)
    
    # Run all test phases
    phases = [
        runner.test_health_checks,
        runner.test_basic_endpoints,
        runner.test_integration_suite,
        runner.test_load_testing
    ]
    
    all_passed = True
    for phase in phases:
        try:
            if not phase():
                all_passed = False
        except Exception as e:
            print(f"❌ Phase failed with error: {e}")
            all_passed = False
    
    # Generate final report
    final_success = runner.generate_report()
    
    # Exit with appropriate code
    sys.exit(0 if final_success else 1)

if __name__ == "__main__":
    main()
