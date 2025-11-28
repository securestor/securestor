 # PyPI Package Compliance and Quality Policies
 # Advanced governance rules for Python package management

 package pypi.compliance

 import rego.v1

 bool_to_num(b) = n if {
     b == true
     n := 1
 }
 bool_to_num(b) = n if {
     b == false
     n := 0
 }

# Quality gates for package promotion
promote_to_stable if {
    # Package must meet all quality criteria
    quality_score >= 85
    security_requirements_met
    documentation_adequate
    test_coverage_sufficient
    dependency_quality_check
}

# Quality scoring system
quality_score := (
    (metadata_completeness * 20) +
    (documentation_score * 15) +
    (test_coverage_score * 25) +
    (dependency_health_score * 20) +
    (security_score * 30) +
    (community_engagement * 10)
) / 12

# Metadata completeness assessment
metadata_completeness := score if {
    required_fields := [
        input.package.name,
        input.package.version,
        input.package.summary,
        input.package.description,
        input.package.author,
        input.package.license,
        input.package.home_page
    ]
    present_fields := count([field | field := required_fields[_]; field != ""])
    score := (present_fields / count(required_fields)) * 100
}

# Documentation quality assessment
documentation_score := score if {
    # Check for README
    has_readme := "README" in [upper(file.name) | file := input.package.files[_]]
    desc_adequate := count(input.package.description) > 100
    has_project_urls := count(input.package.project_urls) > 0
    has_classifiers := count(input.package.classifiers) >= 3
    points := (bool_to_num(has_readme) * 40) + (bool_to_num(desc_adequate) * 30) + (bool_to_num(has_project_urls) * 20) + (bool_to_num(has_classifiers) * 10)
    score := points
}

# Test coverage assessment
test_coverage_score := score if {
    # Check if package has test files
    test_files := [file | 
        file := input.package.files[_]
        contains(lower(file.name), "test")
    ]
    has_tests := count(test_files) > 0
    config_files := ["pytest.ini", "tox.ini", ".coveragerc", "pyproject.toml"]
    has_test_config := count([file |
        file := input.package.files[_]
        file.name in config_files
    ]) > 0
    coverage_pct := input.scan_results.test_coverage.percentage
    base_score := (bool_to_num(has_tests) * 30) + (bool_to_num(has_test_config) * 20) + (coverage_pct / 2)
    score := min(base_score, 100)
}

min(a, b) := a if a <= b
min(a, b) := b if a > b

# Dependency health assessment
dependency_health_score := 100 if {
    deps := input.scan_results.dependencies
    count(deps) == 0
}

dependency_health_score := score if {
    deps := input.scan_results.dependencies
    total_deps := count(deps)
    total_deps > 0

    outdated_deps := count([dep | dep := deps[_]; dep.is_outdated == true])
    vulnerable_deps := count([dep | dep := deps[_]; dep.has_vulnerabilities == true])
    deprecated_deps := count([dep | dep := deps[_]; dep.is_deprecated == true])

    health_ratio := (total_deps - outdated_deps - vulnerable_deps - deprecated_deps) / total_deps
    score := health_ratio * 100
}

# Security score calculation
security_score := s if {
    vulns := input.scan_results.vulnerabilities
    # Base score starts at 100
    base := 100
    # Deduct points based on vulnerabilities
    critical_vulns := count([v | v := vulns[_]; v.severity == "critical"])
    high_vulns := count([v | v := vulns[_]; v.severity == "high"])
    medium_vulns := count([v | v := vulns[_]; v.severity == "medium"])
    low_vulns := count([v | v := vulns[_]; v.severity == "low"])
    deductions := (critical_vulns * 30) + (high_vulns * 20) + (medium_vulns * 10) + (low_vulns * 5)
    # Bonus for security best practices
    bonus := (input.scan_results.has_security_policy * 10) + (input.scan_results.has_code_scanning * 5)
    s := max(0, base - deductions + bonus)
}

max(a, b) := a if a >= b
max(a, b) := b if a < b

# Community engagement score
community_engagement := score if {
    # GitHub stars, forks, issues (if available)
    github_score := input.package.github_metrics.stars / 100  # Scale stars
    download_score := min(input.package.download_stats.monthly / 1000, 20)  # Max 20 points
    recent_updates := input.package.last_updated_days <= 90
    maintenance_score := bool_to_num(recent_updates) * 30
    score := github_score + download_score + maintenance_score
}

# Security requirements validation
security_requirements_met if {
    # No critical or high vulnerabilities
    critical_high_vulns := count([v | 
        v := input.scan_results.vulnerabilities[_]
        v.severity in ["critical", "high"]
    ])
    critical_high_vulns == 0
    
    # License must be compatible
    license_compatible
    
    # No malicious code detected
    input.scan_results.malware.status == "clean"
}

license_compatible if {
    # Check license compatibility matrix
    input.package.license in compatible_licenses
}

license_compatible if {
    # Custom license compatibility check
    license_info := input.scan_results.licenses[_]
    license_info.is_osi_approved == true
    license_info.is_copyleft == false
}

compatible_licenses := {
    "MIT", "BSD", "Apache-2.0", "ISC", "Unlicense",
    "BSD-2-Clause", "BSD-3-Clause", "Apache License 2.0"
}

# Documentation adequacy check
documentation_adequate if {
    # Must have meaningful description
    count(input.package.description) >= 100
    
    # Should have project homepage
    input.package.home_page
    
    # Should have proper README
    has_readme_file
}

has_readme_file if {
    readme_files := [file |
        file := input.package.files[_]
        startswith(upper(file.name), "README")
    ]
    count(readme_files) > 0
}

# Test coverage sufficiency
test_coverage_sufficient if {
    # Must have test files
    has_test_files
    
    # Coverage should be reasonable
    input.scan_results.test_coverage.percentage >= 60
}

has_test_files if {
    test_patterns := ["test_", "_test.py", "tests/", "test/"]
    test_files := [file |
        file := input.package.files[_]
        pattern := test_patterns[_]
        contains(lower(file.name), pattern)
    ]
    count(test_files) > 0
}

# Dependency quality validation
dependency_quality_check if {
    # Dependencies should be up to date
    outdated_critical_deps == 0
    
    # No deprecated dependencies
    deprecated_deps == 0
    
    # Dependency licenses compatible
    dependency_licenses_compatible
}

outdated_critical_deps := count([dep |
    dep := input.scan_results.dependencies[_]
    dep.is_outdated == true
    dep.severity == "high"  # Significantly outdated
])

deprecated_deps := count([dep |
    dep := input.scan_results.dependencies[_] 
    dep.is_deprecated == true
])

dependency_licenses_compatible if {
    incompatible := count([dep |
        dep := input.scan_results.dependencies[_]
        not dep.license in compatible_licenses
        dep.license_category == "copyleft"
    ])
    incompatible == 0
}

# Quarantine policies
require_quarantine if {
    # High risk packages need quarantine
    security_score < 50
}

require_quarantine if {
    # New uploaders with suspicious packages
    suspicious_uploader
    quality_score < 60
}

require_quarantine if {
    # Packages with concerning patterns
    concerning_patterns_detected
}

suspicious_uploader if {
    # New user with high upload volume
    input.user.account_age_days < 30
    input.user.total_packages > 10
}

suspicious_uploader if {
    # User with previous violations
    count(input.user.policy_violations) > 0
}

concerning_patterns_detected if {
    # Package name similar to popular packages (typosquatting)
    potential_typosquat
}

concerning_patterns_detected if {
    # Suspicious metadata patterns
    suspicious_metadata
}

potential_typosquat if {
    # Check against popular package names
    popular_packages := data.popular_pypi_packages
    similarity_threshold := 0.8
    
    # Simple character-based similarity check
    similar_name := [name |
        name := popular_packages[_]
        name != input.package.name
        similarity(input.package.name, name) >= similarity_threshold
    ]
    count(similar_name) > 0
}

# Simplified similarity function (Levenshtein distance approximation)
similarity(s1, s2) := score if {
    len1 := count(s1)
    len2 := count(s2) 
    max_len := max(len1, len2)
    
    # Character overlap scoring (simplified)
    common_chars := count([i |
        i := numbers.range(0, min(len1, len2) - 1)[_]
        s1[i] == s2[i]
    ])
    
    max_len > 0
    score := common_chars / max_len
}

similarity(s1, s2) := score if {
    len1 := count(s1)
    len2 := count(s2)
    max_len := max(len1, len2)
    
    max_len == 0
    score := 0
}

suspicious_metadata if {
    # Generic or meaningless descriptions
    generic_descriptions := [
        "a python package", "python library", "test package",
        "sample package", "example", "demo"
    ]
    
    lower_desc := lower(input.package.description)
    matches := count([desc | 
        desc := generic_descriptions[_]
        contains(lower_desc, desc)
    ])
    matches > 0
}

suspicious_metadata if {
    # Missing or invalid contact information
    not input.package.author_email
}

suspicious_metadata if {
    # Excessive keywords (keyword stuffing)
    count(input.package.keywords) > 15
}

# Automated remediation suggestions
remediation_suggestions := suggestions if {
    suggestions := [suggestion |
        suggestion := generate_suggestions[_]
    ]
}

generate_suggestions contains suggestion if {
    metadata_completeness < 80
    suggestion := {
        "type": "metadata",
        "priority": "medium",
        "message": "Improve package metadata completeness",
        "actions": [
            "Add detailed package description",
            "Include author email and homepage",
            "Add relevant classifiers"
        ]
    }
}

generate_suggestions contains suggestion if {
    documentation_score < 70
    suggestion := {
        "type": "documentation", 
        "priority": "medium",
        "message": "Enhance package documentation",
        "actions": [
            "Add comprehensive README file",
            "Include usage examples",
            "Document API and configuration"
        ]
    }
}

generate_suggestions contains suggestion if {
    test_coverage_score < 60
    suggestion := {
        "type": "testing",
        "priority": "high", 
        "message": "Improve test coverage",
        "actions": [
            "Add unit tests for core functionality", 
            "Include integration tests",
            "Add test configuration files"
        ]
    }
}

generate_suggestions contains suggestion if {
    security_score < 80
    suggestion := {
        "type": "security",
        "priority": "high",
        "message": "Address security concerns", 
        "actions": [
            "Fix identified vulnerabilities",
            "Update vulnerable dependencies",
            "Add security policy documentation"
        ]
    }
}

generate_suggestions contains suggestion if {
    dependency_health_score < 70
    suggestion := {
        "type": "dependencies",
        "priority": "medium",
        "message": "Improve dependency health",
        "actions": [
            "Update outdated dependencies",
            "Remove deprecated packages",
            "Pin dependency versions appropriately"
        ]
    }
}

# Package lifecycle management
lifecycle_action := action if {
    quality_score >= 90
    security_score >= 85
    action := "promote_to_featured"
}

lifecycle_action := action if {
    quality_score >= 75
    security_score >= 80
    action := "promote_to_stable"
}

lifecycle_action := action if {
    quality_score < 50
    action := "quarantine"
}

lifecycle_action := action if {
    security_score < 60
    action := "security_review_required"
}

lifecycle_action := action if {
    require_quarantine
    action := "immediate_quarantine"
}

lifecycle_action := action if {
    not promote_to_stable
    quality_score >= 60
    action := "development_track"
}

# Compliance report generation
compliance_report := {
    "package_id": input.package.id,
    "package_name": input.package.name,
    "version": input.package.version,
    "assessment_timestamp": time.now_ns(),
    "overall_score": quality_score,
    "component_scores": {
        "metadata": metadata_completeness,
        "documentation": documentation_score,
        "testing": test_coverage_score,
        "dependencies": dependency_health_score,
        "security": security_score,
        "community": community_engagement
    },
    "compliance_status": compliance_status,
    "lifecycle_action": lifecycle_action,
    "remediation_suggestions": remediation_suggestions,
    "next_review_date": next_review_timestamp
}

compliance_status := "compliant" if {
    promote_to_stable
    quality_score >= 80
}

compliance_status := "partially_compliant" if {
    not promote_to_stable
    quality_score >= 60
    not require_quarantine
}

compliance_status := "non_compliant" if {
    quality_score < 60
}

compliance_status := "quarantined" if {
    require_quarantine
}

# Next review scheduling
next_review_timestamp := future_time if {
    # High quality packages reviewed less frequently
    quality_score >= 90
    days_ahead := 180  # 6 months
    future_time := time.now_ns() + (days_ahead * 24 * 60 * 60 * 1000000000)
}

next_review_timestamp := future_time if {
    quality_score >= 70
    quality_score < 90
    days_ahead := 90  # 3 months
    future_time := time.now_ns() + (days_ahead * 24 * 60 * 60 * 1000000000)
}

next_review_timestamp := future_time if {
    quality_score < 70
    days_ahead := 30  # 1 month
    future_time := time.now_ns() + (days_ahead * 24 * 60 * 60 * 1000000000)
}

next_review_timestamp := future_time if {
    require_quarantine
    days_ahead := 7  # 1 week
    future_time := time.now_ns() + (days_ahead * 24 * 60 * 60 * 1000000000)
}