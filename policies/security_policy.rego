package securestor.policy

import rego.v1

# Basic security policy for SecureStor artifacts
default allow := false
default action := "deny"

# Allow artifacts with no critical vulnerabilities
allow if {
    input.vulnerabilities.critical == 0
    input.vulnerabilities.high <= 5
}

action := "allow" if {
    allow
}

action := "warn" if {
    input.vulnerabilities.critical == 0
    input.vulnerabilities.high > 5
    input.vulnerabilities.high <= 10
}

action := "quarantine" if {
    input.vulnerabilities.critical > 0
    input.vulnerabilities.critical <= 2
}

action := "block" if {
    input.vulnerabilities.critical > 2
}

# Risk scoring
risk_score := score if {
    score := (input.vulnerabilities.critical * 10) + (input.vulnerabilities.high * 5) + (input.vulnerabilities.medium * 2) + input.vulnerabilities.low
}

risk_level := "low" if {
    risk_score < 10
}

risk_level := "medium" if {
    risk_score >= 10
    risk_score < 50
}

risk_level := "high" if {
    risk_score >= 50
    risk_score < 100
}

risk_level := "critical" if {
    risk_score >= 100
}

reason := msg if {
    allow
    msg := "Artifact meets security requirements"
}

reason := msg if {
    not allow
    action == "warn"
    msg := sprintf("High vulnerability count: %d high, %d critical", [input.vulnerabilities.high, input.vulnerabilities.critical])
}

reason := msg if {
    not allow
    action == "quarantine"
    msg := sprintf("Critical vulnerabilities detected: %d", [input.vulnerabilities.critical])
}

reason := msg if {
    not allow
    action == "block"
    msg := sprintf("Too many critical vulnerabilities: %d", [input.vulnerabilities.critical])
}

# Result structure
result := {
    "allow": allow,
    "action": action,
    "risk_score": risk_score,
    "risk_level": risk_level,
    "reason": reason,
    "timestamp": time.now_ns()
}