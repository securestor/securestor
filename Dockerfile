FROM golang:1.24-alpine3.22 AS builder

WORKDIR /app

# Install dependencies for building and tools
RUN apk update && apk upgrade && apk add --no-cache \
    git \
    curl \
    python3 \
    py3-pip \
    gcc \
    g++ \
    musl-dev \
    python3-dev \
    libffi-dev \
    openssl-dev \
    cargo \
    rust \
    make \
    nodejs \
    npm \
    maven

# Install security tools with version pinning
RUN curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin v1.0.1 && \
    curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin v0.74.0 && \
    curl -sSfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin v0.48.3

# Install additional security tools
RUN curl -sSfL https://github.com/google/osv-scanner/releases/latest/download/osv-scanner_linux_amd64 -o /usr/local/bin/osv-scanner && \
    chmod +x /usr/local/bin/osv-scanner

RUN curl -sSfL https://github.com/trufflesecurity/trufflehog/releases/download/v3.63.2/trufflehog_3.63.2_linux_amd64.tar.gz | \
    tar -xzC /usr/local/bin

# Install cdxgen for SBOM generation
RUN npm install -g @cyclonedx/cdxgen@latest

# Install Python security tools
RUN pip3 install --no-cache-dir --target /opt/python-tools \
    owasp-depscan \
    bandit \
    safety \
    PyYAML \
    requests \
    click \
    wheel \
    setuptools

# Create Python tool wrapper scripts (keep original executables, just add PATH)
RUN mkdir -p /opt/python-tools/bin && \
    echo '#!/bin/bash\nexport PYTHONPATH="/opt/python-tools:$PYTHONPATH"\npython3 -m depscan "$@"' > /opt/python-tools/bin/depscan-wrapper && \
    chmod +x /opt/python-tools/bin/depscan-wrapper

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Runtime stage
FROM alpine:3.20

# Install minimal runtime dependencies including Node.js for cdxgen
RUN apk update && apk upgrade && apk add --no-cache \
    ca-certificates \
    python3 \
    py3-pip \
    nodejs \
    npm \
    maven \
    openjdk11

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy all security scanning tools
COPY --from=builder /usr/local/bin/syft /usr/local/bin/
COPY --from=builder /usr/local/bin/grype /usr/local/bin/
COPY --from=builder /usr/local/bin/trivy /usr/local/bin/
COPY --from=builder /usr/local/bin/osv-scanner /usr/local/bin/
COPY --from=builder /usr/local/bin/trufflehog /usr/local/bin/
COPY --from=builder /opt/python-tools /opt/python-tools

# Ensure cdxgen is available
RUN which cdxgen || npm install -g @cyclonedx/cdxgen@latest

# Add Python tools to PATH and create symlinks
ENV PYTHONPATH="/opt/python-tools"
ENV PATH="/opt/python-tools/bin:/usr/local/bin:$PATH"

# Create symlinks for Python tools and ensure executables
RUN ln -sf /opt/python-tools/bin/depscan /usr/local/bin/depscan && \
    chmod +x /usr/local/bin/syft \
             /usr/local/bin/grype \
             /usr/local/bin/trivy \
             /usr/local/bin/osv-scanner \
             /usr/local/bin/trufflehog \
             /usr/local/bin/depscan

# Create storage directory
RUN mkdir -p /app/storage

# Health check script for security tools
RUN echo '#!/bin/sh' > /usr/local/bin/security-tools-health && \
    echo 'echo "=== Security Tools Health Check ==="' >> /usr/local/bin/security-tools-health && \
    echo 'echo ""' >> /usr/local/bin/security-tools-health && \
    echo 'echo -n "Syft: " && syft --version && echo "✓ OK" || echo "✗ FAILED"' >> /usr/local/bin/security-tools-health && \
    echo 'echo -n "Grype: " && grype version 2>/dev/null | head -n1 && echo "✓ OK" || echo "✗ FAILED"' >> /usr/local/bin/security-tools-health && \
    echo 'echo -n "Trivy: " && trivy --version 2>/dev/null | head -n1 && echo "✓ OK" || echo "✗ FAILED"' >> /usr/local/bin/security-tools-health && \
    echo 'echo -n "OSV-Scanner: " && osv-scanner --version 2>/dev/null && echo "✓ OK" || echo "✗ FAILED"' >> /usr/local/bin/security-tools-health && \
    echo 'echo -n "TruffleHog: " && trufflehog --version 2>/dev/null && echo "✓ OK" || echo "✗ FAILED"' >> /usr/local/bin/security-tools-health && \
    echo 'echo -n "DepScan: " && python3 -c "import depscan; print(\"Available\")" 2>/dev/null && echo "✓ OK" || echo "✗ FAILED"' >> /usr/local/bin/security-tools-health && \
    echo 'echo -n "CDXGen: " && cdxgen --version 2>/dev/null && echo "✓ OK" || echo "✗ FAILED"' >> /usr/local/bin/security-tools-health && \
    echo 'echo ""' >> /usr/local/bin/security-tools-health && \
    echo 'echo "=== All Security Tools Ready ==="' >> /usr/local/bin/security-tools-health && \
    chmod +x /usr/local/bin/security-tools-health

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]