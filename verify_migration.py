#!/usr/bin/env python3
"""
Final verification script for microservices migration.
This script checks that all services and components are properly configured.
"""

import os
import json
import sys
from pathlib import Path

def check_service_structure():
    """Check that all services have required files."""
    services = [
        'auth-gateway', 'tenant-node', 'operation-node', 
        'cbo-engine', 'metadata-catalog', 'query-interpreter', 'monitoring'
    ]
    
    results = {}
    
    for service in services:
        service_path = Path(service)
        results[service] = {
            'exists': service_path.exists(),
            'has_main': (service_path / 'main.py').exists(),
            'has_requirements': (service_path / 'requirements.txt').exists(),
            'has_dockerfile': (service_path / 'Dockerfile').exists(),
            'has_readme': (service_path / 'README.md').exists()
        }
    
    return results

def check_demo_scripts():
    """Check that demo scripts exist."""
    demos = [
        'microservices_demo.py',
        'auth_demo.py', 
        'scaling_demo.py'
    ]
    
    results = {}
    for demo in demos:
        results[demo] = Path(demo).exists()
    
    return results

def check_documentation():
    """Check that documentation is complete."""
    docs = [
        'README.md',
        'QUICKSTART.md',
        'MIGRATION.md',
        'CHANGELOG.md',
        'MIGRATION_COMPLETE.md'
    ]
    
    results = {}
    for doc in docs:
        results[doc] = Path(doc).exists()
    
    return results

def check_infrastructure():
    """Check infrastructure files."""
    infra = [
        'docker-compose.yml',
        'start_services.ps1',
        'start_services.sh',
        'stop_services.sh',
        'cleanup.ps1',
        'cleanup.sh'
    ]
    
    results = {}
    for file in infra:
        results[file] = Path(file).exists()
    
    return results

def check_obsolete_files():
    """Check that obsolete files are gone."""
    obsolete = [
        'main.py',
        'run.py', 
        'config.env.example',
        'requirements.txt',
        'dev-requirements.txt',
        'instructions.txt',
        'setup.bat',
        'setup.sh'
    ]
    
    results = {}
    for file in obsolete:
        results[file] = {
            'exists': Path(file).exists(),
            'should_exist': False
        }
    
    return results

def print_results(title, results, success_key=None):
    """Print formatted results."""
    print(f"\n{'='*60}")
    print(f"  {title}")
    print(f"{'='*60}")
    
    if isinstance(results, dict):
        for key, value in results.items():
            if isinstance(value, dict):
                if success_key:
                    status = "OK" if value.get(success_key, False) else "ERROR"
                else:
                    all_good = all(v for v in value.values() if isinstance(v, bool))
                    status = "OK" if all_good else "WARNING"
                print(f"{status}: {key}")
                for subkey, subvalue in value.items():
                    if isinstance(subvalue, bool):
                        substatus = "OK" if subvalue else "ERROR"
                        print(f"   {substatus}: {subkey}")
            else:
                status = "OK" if value else "ERROR"
                print(f"{status}: {key}")
    else:
        status = "OK" if results else "ERROR"
        print(f"{status}: {title}")

def main():
    """Run all verification checks."""
    print("MICROSERVICES MIGRATION VERIFICATION")
    print("=" * 60)
    
    # Check services
    service_results = check_service_structure()
    print_results("MICROSERVICES STRUCTURE", service_results)
    
    # Check demos
    demo_results = check_demo_scripts()
    print_results("DEMO SCRIPTS", demo_results)
    
    # Check documentation
    doc_results = check_documentation()
    print_results("DOCUMENTATION", doc_results)
    
    # Check infrastructure
    infra_results = check_infrastructure()
    print_results("INFRASTRUCTURE", infra_results)
    
    # Check obsolete files (these should NOT exist)
    obsolete_results = check_obsolete_files()
    print(f"\n{'='*60}")
    print(f"  OBSOLETE FILES (Should NOT exist)")
    print(f"{'='*60}")
    
    obsolete_found = False
    for file, info in obsolete_results.items():
        if info['exists']:
            print(f"WARNING: {file} - STILL EXISTS (should be deleted)")
            obsolete_found = True
        else:
            print(f"OK: {file} - Properly removed")
    
    # Summary
    print(f"\n{'='*60}")
    print(f"  MIGRATION SUMMARY")
    print(f"{'='*60}")
    
    total_services = len(service_results)
    complete_services = sum(1 for s in service_results.values() 
                          if all(v for v in s.values() if isinstance(v, bool)))
    
    total_demos = len(demo_results)
    working_demos = sum(demo_results.values())
    
    total_docs = len(doc_results)
    complete_docs = sum(doc_results.values())
    
    total_infra = len(infra_results)
    working_infra = sum(infra_results.values())
    
    print(f"Services: {complete_services}/{total_services} complete")
    print(f"Demos: {working_demos}/{total_demos} available")
    print(f"Documentation: {complete_docs}/{total_docs} files")
    print(f"Infrastructure: {working_infra}/{total_infra} files")
    print(f"Obsolete files: {'None found' if not obsolete_found else 'Some remain'}")
    
    # Final status
    all_good = (
        complete_services == total_services and
        working_demos == total_demos and
        complete_docs == total_docs and
        working_infra == total_infra and
        not obsolete_found
    )
    
    if all_good:
        print(f"\nMIGRATION COMPLETE - ALL CHECKS PASSED!")
        print(f"Ready for production deployment.")
        return 0
    else:
        print(f"\nMIGRATION INCOMPLETE - SOME ISSUES FOUND")
        print(f"Review the items marked with 'ERROR' above.")
        return 1

if __name__ == "__main__":
    sys.exit(main())
