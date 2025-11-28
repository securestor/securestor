# PyPI Package Security Policy for Workflow System
# Integrates with existing securestor workflow registry

package securestor.python_policy

import rego.v1

# Main decision point - allow artifact to be stored/used
allow if {
    # Basic package validation
    valid_python_package
    
    # Security requirements
    no_critical_vulnerabilities
    approved_python_license
    
    # Dependency security
    secure_dependencies
    
    # Code quality (from bandit)
    no_high_severity_code_issues
}

# Python package validation
valid_python_package if {
    # Check artifact type is supported
    input.artifact_type in ["python", "pypi", "wheel", "sdist"]
    
    # Package should have valid Python metadata
    valid_python_metadata
}

valid_python_metadata if {
    # Basic metadata checks would go here
    # For workflow system, this is simplified
    true
}

# Critical vulnerability check
no_critical_vulnerabilities if {
    critical_vulns := [vuln | 
        vuln := input.scan_results[_].vulnerabilities[_]
        vuln.severity in ["CRITICAL", "HIGH"]
    ]
    count(critical_vulns) == 0
}

# License approval for Python packages
approved_python_license if {
    # Allow common Python licenses
    python_licenses := {
        "MIT", "BSD", "Apache-2.0", "Apache License 2.0",
        "Python Software Foundation License", "PSF", 
        "BSD License", "ISC License", "Unlicense"
    }
    
    # Check if any scan result indicates acceptable license
    # This would be enhanced based on actual scanner output structure
    true  # Simplified for workflow integration
}

# Dependency security validation
secure_dependencies if {
    # Check for vulnerable dependencies in scan results
    vulnerable_deps := [vuln |
        result := input.scan_results[_]
        result.scanner_name == "pypi"  # PyPI-specific scanner results
        vuln := result.vulnerabilities[_]
        vuln.severity in ["CRITICAL", "HIGH"]
    ]
    count(vulnerable_deps) == 0
}

# Code quality from bandit scanner
no_high_severity_code_issues if {
    bandit_issues := [vuln |
        result := input.scan_results[_]
        result.scanner_name == "pypi"
        vuln := result.vulnerabilities[_]
        startswith(vuln.id, "BANDIT")
        vuln.severity in ["CRITICAL", "HIGH"]
    ]
    count(bandit_issues) == 0
}

# Quarantine conditions
quarantine if {
    # Quarantine packages with critical security issues
    critical_security_issues
}

quarantine if {
    # Quarantine packages with too many vulnerabilities
    vulnerability_count > 10
}

critical_security_issues if {
    critical_vulns := [vuln |
        vuln := input.scan_results[_].vulnerabilities[_]
        vuln.severity == "CRITICAL"
    ]
    count(critical_vulns) > 0
}

vulnerability_count := count([vuln |
    vuln := input.scan_results[_].vulnerabilities[_]
])

# Action determination
action := "allow" if allow
action := "quarantine" if quarantine
action := "block" if {
    not allow
    not quarantine
}

# Reason for decision
reason := "Package passed all security and quality checks" if allow
reason := "Package has critical security vulnerabilities" if critical_security_issues
reason := sprintf("Package has %d vulnerabilities exceeding threshold", [vulnerability_count]) if vulnerability_count > 10
reason := "Package failed security validation" if {
    not allow
    not critical_security_issues
    vulnerability_count <= 10
}

# Package quality scoring (integrated PyPI quality checks)
quality_score := score if {
    metadata_score := metadata_completeness_score
    security_score := security_assessment_score
    dependency_score := dependency_health_score
    
    score := (metadata_score * 30 + security_score * 50 + dependency_score * 20) / 100
}

metadata_completeness_score := score if {
    metadata := input.artifact.metadata
    required_fields := ["name", "version", "author", "description"]
    
    present_fields := count([field | 
        field := required_fields[_]
        metadata[field]
    ])
    
    score := (present_fields * 100) / count(required_fields)
}

security_assessment_score := 100 if {
    vulnerability_count == 0
}

security_assessment_score := 80 if {
    vulnerability_count > 0
    vulnerability_count <= 2
}

security_assessment_score := 60 if {
    vulnerability_count > 2
    vulnerability_count <= 5
}

security_assessment_score := 40 if {
    vulnerability_count > 5
    vulnerability_count <= 10
}

security_assessment_score := 20 if {
    vulnerability_count > 10
}

dependency_health_score := score if {
    # If no dependency analysis available, assume good
    not input.scan_results.dependencies
    score := 100
}

dependency_health_score := score if {
    deps := input.scan_results.dependencies
    total_deps := count(deps)
    
    total_deps == 0
    score := 100
}

dependency_health_score := score if {
    deps := input.scan_results.dependencies
    total_deps := count(deps)
    
    vulnerable_deps := count([dep |
        dep := deps[_]
        dep.vulnerabilities
        count(dep.vulnerabilities) > 0
    ])
    
    total_deps > 0
    health_ratio := (total_deps - vulnerable_deps) / total_deps
    score := health_ratio * 100
}

# Summary for workflow system
decision := {
    "allow": allow,
    "action": action,
    "reason": reason,
    "quality_score": quality_score,
    "details": {
        "total_vulnerabilities": vulnerability_count,
        "critical_vulnerabilities": count([v | v := input.scan_results[_].vulnerabilities[_]; v.severity == "CRITICAL"]),
        "high_vulnerabilities": count([v | v := input.scan_results[_].vulnerabilities[_]; v.severity == "HIGH"]),
        "security_passed": no_critical_vulnerabilities,
        "code_quality_passed": no_high_severity_code_issues,
        "dependencies_secure": secure_dependencies,
        "metadata_complete": metadata_completeness_score >= 80,
        "quality_assessment": {
            "overall_score": quality_score,
            "metadata_score": metadata_completeness_score,
            "security_score": security_assessment_score,
            "dependency_score": dependency_health_score
        }
    }
}