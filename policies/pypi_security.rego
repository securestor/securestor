# PyPI Package Security and Governance Policies
# Open Policy Agent (OPA) rules for Python package management

package pypi.security

import rego.v1

# Allow package upload if all security checks pass
allow_upload if {
    # Package must pass basic validation
    valid_package_name
    valid_version_format
    
    # Security requirements
    no_critical_vulnerabilities
    approved_license
    dependency_security_check
    
    # Governance requirements
    authorized_uploader
    repository_access_allowed
}

# Package name validation (PEP 508)
valid_package_name if {
    # Package name must be valid according to PEP 508
    regex.match(`^([A-Z0-9]|[A-Z0-9][A-Z0-9._-]*[A-Z0-9])$`, upper(input.package.name))
    
    # Name must not be reserved
    not reserved_package_name
    
    # Length constraints
    count(input.package.name) <= 214
    count(input.package.name) >= 1
}

# Reserved package names that should not be allowed
reserved_package_name if {
    lower(input.package.name) in {
        "pip", "setuptools", "wheel", "python", "stdlib",
        "admin", "administrator", "root", "test", "tests",
        "www", "ftp", "mail", "email", "api", "app"
    }
}

# Version format validation (PEP 440)
valid_version_format if {
    # Basic version pattern validation
    regex.match(`^([1-9][0-9]*!)?(0|[1-9][0-9]*)(\.(0|[1-9][0-9]*))*((a|b|rc)(0|[1-9][0-9]*))?(\.post(0|[1-9][0-9]*))?(\.dev(0|[1-9][0-9]*))?$`, input.package.version)
}

# Critical vulnerability check
no_critical_vulnerabilities if {
    count(input.scan_results.vulnerabilities) == 0
} else if {
    critical_vulns := [vuln | vuln := input.scan_results.vulnerabilities[_]; vuln.severity in ["critical", "high"]]
    count(critical_vulns) == 0
}

# License approval check
approved_license if {
    # Package must have a license
    input.package.license
    
    # License must be in approved list or meet criteria
    license_approved
}

license_approved if {
    lower(input.package.license) in approved_licenses
}

license_approved if {
    # Allow if license is permissive and not copyleft
    license_info := input.scan_results.licenses[_]
    license_info.category == "permissive"
    license_info.is_approved == true
}

# Approved license list
approved_licenses := {
    "mit", "bsd", "apache", "apache-2.0", "apache 2.0",
    "isc", "unlicense", "public domain", "cc0", "cc0-1.0"
}

# Dependency security validation
dependency_security_check if {
    # All dependencies must be secure
    vulnerable_deps := [dep | 
        dep := input.scan_results.dependencies[_]
        dep_vulnerable(dep)
    ]
    count(vulnerable_deps) == 0
}

dep_vulnerable(dep) if {
    # Check if dependency has known vulnerabilities
    vuln := input.scan_results.vulnerabilities[_]
    vuln.package == dep.name
    vuln.severity in ["high", "critical"]
}

# Uploader authorization
authorized_uploader if {
    # User must be authenticated
    input.user.id
    
    # User must have upload permission for this repository
    user_has_permission("pypi:upload")
    
    # User must not be blacklisted
    not user_blacklisted
}

user_has_permission(permission) if {
    input.user.permissions[_] == permission
}

user_has_permission(permission) if {
    input.user.permissions[_] == "admin"
}

user_blacklisted if {
    input.user.id in data.blacklisted_users
}

# Repository access control
repository_access_allowed if {
    # Repository must exist and be active
    input.repository.is_active == true
    
    # User must have access to this repository
    repository_permission_check
    
    # Repository must allow PyPI packages
    "pypi" in input.repository.supported_types
}

repository_permission_check if {
    # Public repositories allow uploads from authenticated users
    input.repository.public_access == true
    input.user.id
}

repository_permission_check if {
    # Private repositories require explicit permission
    input.repository.public_access == false
    user_has_repository_access
}

user_has_repository_access if {
    input.user.id in input.repository.authorized_users
}

user_has_repository_access if {
    user_role := input.user.roles[_]
    user_role.name in ["admin", "maintainer"]
}

# Package deletion policy
allow_delete if {
    # Only allow deletion within time window
    deletion_time_window_valid
    
    # User must be authorized
    delete_authorization_check
    
    # Package must not have dependencies
    no_active_dependencies
}

deletion_time_window_valid if {
    # Allow deletion within 72 hours of upload
    upload_time := time.parse_rfc3339_ns(input.package.upload_time)
    current_time := time.now_ns()
    time_diff := current_time - upload_time
    time_diff <= 259200000000000  # 72 hours in nanoseconds
}

delete_authorization_check if {
    # Package uploader can delete
    input.user.id == input.package.uploaded_by
}

delete_authorization_check if {
    # Repository admin can delete
    user_has_permission("pypi:delete")
}

no_active_dependencies if {
    # Check if other packages depend on this one
    count(input.package.dependents) == 0
}

# Content scanning requirements
content_scan_required if {
    # Large packages must be scanned
    input.package.file_size > 10485760  # 10MB
}

content_scan_required if {
    # Executable files require scanning
    has_executable_content
}

has_executable_content if {
    # Check for executable file patterns
    file := input.package.files[_]
    regex.match(`\.(exe|bat|sh|py|js)$`, lower(file.name))
}

# Malware detection
no_malware_detected if {
    # No malicious patterns detected
    not malicious_content_found
    
    # Clean scan results
    malware_scan_clean
}

malicious_content_found if {
    # Check for suspicious patterns
    pattern := suspicious_patterns[_]
    contains(input.package.description, pattern)
}

malicious_content_found if {
    # Check for encoded content
    has_base64_encoded_content
}

suspicious_patterns := [
    "bitcoin", "cryptocurrency", "mining", "wallet",
    "password stealer", "keylogger", "backdoor",
    "reverse shell", "payload", "exploit"
]

has_base64_encoded_content if {
    # Simple base64 detection in package content
    regex.match(`[A-Za-z0-9+/]{20,}={0,2}`, input.package.content)
}

malware_scan_clean if {
    input.scan_results.malware.status == "clean"
}

# Version management policies
version_increment_valid if {
    # New version must be greater than existing versions
    not version_already_exists
    semantic_version_valid
}

version_already_exists if {
    existing_version := input.package.existing_versions[_]
    existing_version == input.package.version
}

semantic_version_valid if {
    # Version should follow semantic versioning principles
    # This is a simplified check
    parts := split(input.package.version, ".")
    count(parts) >= 2
    count(parts) <= 4
}

# File type restrictions
allowed_file_types if {
    file := input.package.files[_]
    file_extension := lower(substring(file.name, indexof(file.name, "."), -1))
    file_extension in {".whl", ".tar.gz", ".zip"}
}

# Size limitations
within_size_limits if {
    # Individual file size limit (100MB)
    file := input.package.files[_]
    file.size <= 104857600
    
    # Total package size limit (500MB)
    total_size <= 524288000
}

total_size := sum([file.size | file := input.package.files[_]])

# Metadata completeness
complete_metadata if {
    # Required fields must be present
    input.package.name
    input.package.version
    input.package.summary
    input.package.author
    
    # Description should be meaningful
    count(input.package.description) > 20
}

# Rate limiting
within_upload_rate_limits if {
    # User upload rate limit (10 packages per hour)
    user_uploads_last_hour <= 10
    
    # Repository upload rate limit (100 packages per hour)
    repo_uploads_last_hour <= 100
}

user_uploads_last_hour := count([upload |
    upload := data.recent_uploads[_]
    upload.user_id == input.user.id
    upload.timestamp > (time.now_ns() - 3600000000000)  # 1 hour
])

repo_uploads_last_hour := count([upload |
    upload := data.recent_uploads[_]
    upload.repository_id == input.repository.id
    upload.timestamp > (time.now_ns() - 3600000000000)  # 1 hour
])

# Compliance aggregation
compliance_status := {
    "package_validation": {
        "valid_name": valid_package_name,
        "valid_version": valid_version_format,
        "complete_metadata": complete_metadata
    },
    "security": {
        "no_critical_vulnerabilities": no_critical_vulnerabilities,
        "approved_license": approved_license,
        "dependency_security": dependency_security_check,
        "no_malware": no_malware_detected
    },
    "governance": {
        "authorized_uploader": authorized_uploader,
        "repository_access": repository_access_allowed,
        "within_rate_limits": within_upload_rate_limits
    },
    "content": {
        "allowed_file_types": allowed_file_types,
        "within_size_limits": within_size_limits
    }
}

# Helper function to convert boolean to number
bool_to_int(true) := 1
bool_to_int(false) := 0

# Overall compliance score
compliance_score := (
    (bool_to_int(compliance_status.package_validation.valid_name) * 10) +
    (bool_to_int(compliance_status.package_validation.valid_version) * 10) +
    (bool_to_int(compliance_status.package_validation.complete_metadata) * 5) +
    (bool_to_int(compliance_status.security.no_critical_vulnerabilities) * 25) +
    (bool_to_int(compliance_status.security.approved_license) * 15) +
    (bool_to_int(compliance_status.security.dependency_security) * 20) +
    (bool_to_int(compliance_status.security.no_malware) * 30) +
    (bool_to_int(compliance_status.governance.authorized_uploader) * 20) +
    (bool_to_int(compliance_status.governance.repository_access) * 15) +
    (bool_to_int(compliance_status.governance.within_rate_limits) * 5) +
    (bool_to_int(compliance_status.content.allowed_file_types) * 10) +
    (bool_to_int(compliance_status.content.within_size_limits) * 5)
) / 17

# Policy violations
violations := [violation |
    violation := policy_violations[_]
]

policy_violations contains violation if {
    not valid_package_name
    violation := {
        "type": "package_validation",
        "severity": "high",
        "message": "Invalid package name format",
        "code": "PYPI001"
    }
}

policy_violations contains violation if {
    not valid_version_format
    violation := {
        "type": "package_validation", 
        "severity": "high",
        "message": "Invalid version format (PEP 440)",
        "code": "PYPI002"
    }
}

policy_violations contains violation if {
    not no_critical_vulnerabilities
    violation := {
        "type": "security",
        "severity": "critical", 
        "message": "Package contains critical vulnerabilities",
        "code": "PYPI003"
    }
}

policy_violations contains violation if {
    not approved_license
    violation := {
        "type": "compliance",
        "severity": "medium",
        "message": "Package license not approved", 
        "code": "PYPI004"
    }
}

policy_violations contains violation if {
    not authorized_uploader
    violation := {
        "type": "authorization",
        "severity": "critical",
        "message": "User not authorized to upload packages",
        "code": "PYPI005"
    }
}

policy_violations contains violation if {
    not within_size_limits
    violation := {
        "type": "content",
        "severity": "medium",
        "message": "Package exceeds size limits",
        "code": "PYPI006"
    }
}

policy_violations contains violation if {
    not allowed_file_types
    violation := {
        "type": "content",
        "severity": "high", 
        "message": "Package contains disallowed file types",
        "code": "PYPI007"
    }
}

policy_violations contains violation if {
    malicious_content_found
    violation := {
        "type": "security",
        "severity": "critical",
        "message": "Malicious content detected in package",
        "code": "PYPI008"
    }
}

# Final decision
decision := {
    "allow": allow_upload,
    "compliance_score": compliance_score,
    "violations": violations,
    "status": policy_status
}

policy_status := "approved" if {
    allow_upload
    compliance_score >= 80
    count(violations) == 0
}

policy_status := "conditional" if {
    not allow_upload
    compliance_score >= 60
    count([v | v := violations[_]; v.severity == "critical"]) == 0
}

policy_status := "rejected" if {
    not allow_upload
    compliance_score < 60
}

policy_status := "rejected" if {
    count([v | v := violations[_]; v.severity == "critical"]) > 0
}