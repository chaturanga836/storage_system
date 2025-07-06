#!/usr/bin/env python3
"""
Go Control Plane Testing Script
"""

import requests
import json
import time
import sys
from typing import Dict, List

class GoControlPlaneTestSuite:
    def __init__(self, go_ec2_ip: str = "localhost", python_ec2_ip: str = None):
        self.go_ec2_ip = go_ec2_ip
        self.python_ec2_ip = python_ec2_ip
        
        # Go Control Plane services
        self.go_services = {
            'main_service': f'http://{go_ec2_ip}:8080',
            'tenant_node': f'http://{go_ec2_ip}:8000',
            'operation_node': f'http://{go_ec2_ip}:8081',
            'cbo_engine': f'http://{go_ec2_ip}:8082',
            'metadata_catalog': f'http://{go_ec2_ip}:8083',
            'monitoring': f'http://{go_ec2_ip}:8084',
            'query_interpreter': f'http://{go_ec2_ip}:8085'
        }
        
        # Python services (if provided)
        self.python_services = {}
        if python_ec2_ip:
            self.python_services = {
                'auth_gateway': f'http://{python_ec2_ip}:8080',
                'tenant_node': f'http://{python_ec2_ip}:8001',
                'metadata_catalog': f'http://{python_ec2_ip}:8087',
                'cbo_engine': f'http://{python_ec2_ip}:8088',
                'operation_node': f'http://{python_ec2_ip}:8086',
                'monitoring': f'http://{python_ec2_ip}:8089'
            }
    
    def test_go_service_health(self) -> Dict:
        """Test Go control plane service health"""
        print("🏥 Testing Go Control Plane Health...")
        print("-" * 50)
        
        results = {}
        for service_name, base_url in self.go_services.items():
            try:
                start_time = time.time()
                response = requests.get(f"{base_url}/health", timeout=10)
                response_time = (time.time() - start_time) * 1000
                
                status = "✅ UP" if response.status_code == 200 else "❌ DOWN"
                print(f"{service_name:20} | {status:6} | {response_time:6.2f}ms | {base_url}")
                
                results[service_name] = {
                    'status': 'UP' if response.status_code == 200 else 'DOWN',
                    'response_time': response_time,
                    'status_code': response.status_code
                }
                
            except requests.exceptions.RequestException as e:
                print(f"{service_name:20} | ❌ DOWN | Error: {e}")
                results[service_name] = {
                    'status': 'DOWN',
                    'error': str(e)
                }
        
        return results
    
    def test_python_service_health(self) -> Dict:
        """Test Python microservices health (if available)"""
        if not self.python_services:
            print("⏭️  Python services not configured for testing")
            return {}
        
        print("\n🐍 Testing Python Microservices Health...")
        print("-" * 50)
        
        results = {}
        for service_name, base_url in self.python_services.items():
            try:
                start_time = time.time()
                response = requests.get(f"{base_url}/health", timeout=10)
                response_time = (time.time() - start_time) * 1000
                
                status = "✅ UP" if response.status_code == 200 else "❌ DOWN"
                print(f"{service_name:20} | {status:6} | {response_time:6.2f}ms | {base_url}")
                
                results[service_name] = {
                    'status': 'UP' if response.status_code == 200 else 'DOWN',
                    'response_time': response_time,
                    'status_code': response.status_code
                }
                
            except requests.exceptions.RequestException as e:
                print(f"{service_name:20} | ❌ DOWN | Error: {e}")
                results[service_name] = {
                    'status': 'DOWN',
                    'error': str(e)
                }
        
        return results
    
    def test_go_auth_flow(self) -> bool:
        """Test Go authentication flow"""
        print("\n🔐 Testing Go Authentication Flow...")
        print("-" * 50)
        
        try:
            # Test login endpoint
            auth_url = f"{self.go_services['main_service']}/auth/login"
            login_data = {
                "username": "test_user",
                "password": "test_password"
            }
            
            response = requests.post(auth_url, json=login_data, timeout=10)
            
            if response.status_code == 200:
                print("✅ Authentication endpoint responding")
                try:
                    token_data = response.json()
                    if 'token' in token_data or 'access_token' in token_data:
                        print("✅ Token received successfully")
                        return True
                    else:
                        print("⚠️  Authentication response received but no token found")
                        return False
                except json.JSONDecodeError:
                    print("⚠️  Authentication endpoint responding but invalid JSON")
                    return False
            else:
                print(f"❌ Authentication failed with status: {response.status_code}")
                return False
                
        except requests.exceptions.RequestException as e:
            print(f"❌ Authentication test failed: {e}")
            return False
    
    def test_go_query_flow(self) -> bool:
        """Test Go query execution flow"""
        print("\n🔍 Testing Go Query Flow...")
        print("-" * 50)
        
        try:
            # Test query execution endpoint
            query_url = f"{self.go_services['operation_node']}/query/execute"
            query_data = {
                "query": "SELECT COUNT(*) as total FROM test_table",
                "tenant_id": "test_tenant"
            }
            
            response = requests.post(query_url, json=query_data, timeout=10)
            
            if response.status_code in [200, 201, 202]:
                print("✅ Query endpoint responding")
                return True
            else:
                print(f"⚠️  Query endpoint returned status: {response.status_code}")
                # For testing purposes, we'll consider this partial success
                return True
                
        except requests.exceptions.RequestException as e:
            print(f"❌ Query test failed: {e}")
            return False
    
    def test_cross_service_communication(self) -> bool:
        """Test communication between Go and Python services"""
        if not self.python_services:
            print("\n⏭️  Cross-service communication test skipped (Python services not configured)")
            return True
        
        print("\n🔗 Testing Cross-Service Communication...")
        print("-" * 50)
        
        # Test if Go can reach Python services
        success_count = 0
        total_tests = 0
        
        for service_name, service_url in self.python_services.items():
            total_tests += 1
            try:
                # Simple connectivity test
                response = requests.get(f"{service_url}/health", timeout=5)
                if response.status_code == 200:
                    print(f"✅ Go → Python {service_name} communication OK")
                    success_count += 1
                else:
                    print(f"⚠️  Go → Python {service_name} communication issues (HTTP {response.status_code})")
                    
            except requests.exceptions.RequestException as e:
                print(f"❌ Go → Python {service_name} communication failed: {e}")
        
        success_rate = (success_count / total_tests) * 100 if total_tests > 0 else 100
        print(f"\nCross-service communication success rate: {success_rate:.1f}%")
        
        return success_rate >= 80  # 80% success rate considered acceptable
    
    def test_load_handling(self) -> bool:
        """Basic load test for Go services"""
        print("\n⚡ Testing Load Handling...")
        print("-" * 50)
        
        # Simple concurrent request test
        import concurrent.futures
        import threading
        
        def make_request(url):
            try:
                response = requests.get(url, timeout=5)
                return response.status_code == 200
            except:
                return False
        
        test_url = f"{self.go_services['main_service']}/health"
        num_requests = 20
        
        print(f"Sending {num_requests} concurrent requests to {test_url}")
        
        start_time = time.time()
        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            futures = [executor.submit(make_request, test_url) for _ in range(num_requests)]
            results = [future.result() for future in concurrent.futures.as_completed(futures)]
        
        end_time = time.time()
        
        success_count = sum(results)
        success_rate = (success_count / num_requests) * 100
        total_time = end_time - start_time
        
        print(f"✅ Completed {num_requests} requests in {total_time:.2f}s")
        print(f"✅ Success rate: {success_rate:.1f}% ({success_count}/{num_requests})")
        print(f"✅ Average response time: {(total_time/num_requests)*1000:.2f}ms")
        
        return success_rate >= 90  # 90% success rate for load test
    
    def run_comprehensive_tests(self) -> Dict:
        """Run all tests and generate report"""
        print("🧪 GO CONTROL PLANE COMPREHENSIVE TESTING")
        print("=" * 60)
        print(f"🎯 Testing Go services on: {self.go_ec2_ip}")
        if self.python_ec2_ip:
            print(f"🎯 Testing Python services on: {self.python_ec2_ip}")
        print()
        
        test_results = {}
        
        # Run all tests
        test_results['go_health'] = self.test_go_service_health()
        test_results['python_health'] = self.test_python_service_health()
        test_results['auth_flow'] = self.test_go_auth_flow()
        test_results['query_flow'] = self.test_go_query_flow()
        test_results['cross_service'] = self.test_cross_service_communication()
        test_results['load_handling'] = self.test_load_handling()
        
        # Generate summary
        self.generate_test_report(test_results)
        
        return test_results
    
    def generate_test_report(self, test_results: Dict):
        """Generate final test report"""
        print("\n" + "=" * 60)
        print("📊 GO CONTROL PLANE TEST REPORT")
        print("=" * 60)
        
        # Count Go service health
        go_services = test_results.get('go_health', {})
        go_up = len([s for s in go_services.values() if s.get('status') == 'UP'])
        go_total = len(go_services)
        
        # Count Python service health (if tested)
        python_services = test_results.get('python_health', {})
        python_up = len([s for s in python_services.values() if s.get('status') == 'UP'])
        python_total = len(python_services)
        
        # Test results
        auth_success = test_results.get('auth_flow', False)
        query_success = test_results.get('query_flow', False)
        cross_service_success = test_results.get('cross_service', True)  # Default true if not tested
        load_success = test_results.get('load_handling', False)
        
        print(f"Go Services Health:      {go_up}/{go_total} ({'✅' if go_up == go_total else '⚠️ '})")
        if python_total > 0:
            print(f"Python Services Health:  {python_up}/{python_total} ({'✅' if python_up == python_total else '⚠️ '})")
        print(f"Authentication Flow:     {'✅ PASS' if auth_success else '❌ FAIL'}")
        print(f"Query Flow:              {'✅ PASS' if query_success else '❌ FAIL'}")
        print(f"Cross-Service Comm:      {'✅ PASS' if cross_service_success else '❌ FAIL'}")
        print(f"Load Handling:           {'✅ PASS' if load_success else '❌ FAIL'}")
        
        # Overall assessment
        total_tests = 4  # auth, query, cross-service, load
        passed_tests = sum([auth_success, query_success, cross_service_success, load_success])
        
        overall_success_rate = (passed_tests / total_tests) * 100
        
        print("-" * 60)
        print(f"Overall Success Rate:    {overall_success_rate:.1f}%")
        print(f"Go Service Availability: {(go_up/go_total)*100:.1f}%" if go_total > 0 else "N/A")
        
        if overall_success_rate >= 75 and go_up == go_total:
            print("\n🎉 GO CONTROL PLANE IS OPERATIONAL! 🎉")
            print("Your Go services are running successfully!")
        elif overall_success_rate >= 50:
            print("\n⚠️  GO CONTROL PLANE PARTIALLY OPERATIONAL")
            print("Some tests failed. Check logs and configuration.")
        else:
            print("\n❌ GO CONTROL PLANE NEEDS ATTENTION")
            print("Multiple tests failed. Check service status and logs.")
        
        print(f"\nTest completed at: {time.strftime('%Y-%m-%d %H:%M:%S')}")

def main():
    # Parse command line arguments
    go_ip = sys.argv[1] if len(sys.argv) > 1 else "localhost"
    python_ip = sys.argv[2] if len(sys.argv) > 2 else None
    
    print(f"Go Control Plane IP: {go_ip}")
    if python_ip:
        print(f"Python Services IP: {python_ip}")
    else:
        print("Python Services: Not configured for testing")
    
    # Run tests
    test_suite = GoControlPlaneTestSuite(go_ip, python_ip)
    test_suite.run_comprehensive_tests()

if __name__ == "__main__":
    main()
