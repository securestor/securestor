# Contributing to SecureStor

We love your input! We want to make contributing to SecureStor as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## 🚀 Development Process

We use GitHub to host code, track issues and feature requests, as well as accept pull requests.

## 📋 Pull Request Process

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.
6. Issue that pull request!

## 🛠️ Development Setup

### Prerequisites

- **Go 1.21+** - Backend development
- **Node.js 18+** - Frontend development
- **Docker & Docker Compose** - Local development environment
- **PostgreSQL 14+** - Database
- **Redis 7+** - Caching and sessions
- **Git** - Version control

### Local Development Setup

```bash
# 1. Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/securestor.git
cd securestor

# 2. Start development dependencies
docker-compose -f docker-compose.dev.yml up -d postgres redis

# 3. Install backend dependencies
go mod download

# 4. Install frontend dependencies
cd frontend
npm install
cd ..

# 5. Copy environment configuration
cp .env.example .env.dev
# Edit .env.dev with development settings

# 6. Run database migrations
go run cmd/migrate/main.go up

# 7. Start backend (in one terminal)
go run cmd/api/main.go

# 8. Start frontend (in another terminal)
cd frontend
npm start
```

### Environment Configuration

Your `.env.dev` should include:

```bash
# Database
DATABASE_URL=postgres://securestor:securestor@localhost:5432/securestor?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# JWT
JWT_SECRET=dev-secret-change-in-production

# Environment
ENVIRONMENT=development
LOG_LEVEL=debug

# Frontend
FRONTEND_URL=http://localhost:3000
API_BASE_URL=http://localhost:8080
```

## 🧪 Testing

### Backend Tests

```bash
# Run unit tests
go test ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run integration tests
go test -tags=integration ./...

# Run specific test
go test -run TestArtifactUpload ./internal/handlers
```

### Frontend Tests

```bash
cd frontend

# Run unit tests
npm test

# Run tests with coverage
npm run test:coverage

# Run e2e tests
npm run test:e2e
```

### API Testing

```bash
# Install Newman (Postman CLI)
npm install -g newman

# Run API tests
newman run tests/api/securestor.postman_collection.json \
  -e tests/api/dev.postman_environment.json
```

## 📝 Code Style

### Go Code Style

We follow standard Go conventions:

- Use `gofmt` for formatting
- Use `golint` for linting
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use meaningful variable and function names
- Add comments for exported functions and types

```bash
# Format code
gofmt -w .

# Run linter
golangci-lint run

# Check for common issues
go vet ./...
```

### Frontend Code Style

We use Prettier and ESLint:

```bash
cd frontend

# Format code
npm run format

# Lint code
npm run lint

# Fix linting issues
npm run lint:fix
```

### Commit Message Format

We use conventional commits:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

**Examples:**

```
feat(auth): add OAuth2 authentication support

fix(storage): resolve race condition in artifact upload

docs(api): update authentication endpoint documentation

test(scanner): add unit tests for vulnerability detection
```

## 🐛 Bug Reports

We use GitHub issues to track public bugs. Report a bug by [opening a new issue](https://github.com/securestor/securestor/issues/new).

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

### Bug Report Template

```markdown
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Screenshots**
If applicable, add screenshots to help explain your problem.

**Environment:**
 - OS: [e.g. Ubuntu 20.04]
 - SecureStor Version: [e.g. v0.1.0]
 - Docker Version: [e.g. 20.10.21]
 - Browser: [e.g. Chrome 108]

**Additional context**
Add any other context about the problem here.
```

## 💡 Feature Requests

We use GitHub Discussions for feature requests. Before creating a new request:

1. Check if a similar request already exists
2. Clearly describe the problem you're trying to solve
3. Explain why this feature would be useful to SecureStor users
4. Consider potential implementation approaches

## 🏷️ Issue Labels

We use labels to categorize issues:

- `bug`: Something isn't working
- `enhancement`: New feature or request
- `documentation`: Improvements or additions to documentation
- `good first issue`: Good for newcomers
- `help wanted`: Extra attention is needed
- `priority/high`: High priority issue
- `priority/medium`: Medium priority issue
- `priority/low`: Low priority issue
- `area/backend`: Backend related
- `area/frontend`: Frontend related
- `area/security`: Security related
- `area/performance`: Performance related

## 📚 Documentation

Documentation improvements are always welcome! This includes:

- API documentation
- Code comments
- README improvements
- Tutorial and guide creation
- Architecture documentation

### Documentation Guidelines

- Use clear, concise language
- Include code examples where appropriate
- Update documentation when changing functionality
- Follow the existing documentation structure

## 🔒 Security

### Reporting Security Issues

**DO NOT** create GitHub issues for security vulnerabilities. Instead:

1. Email security@securestor.io with details
2. Include steps to reproduce the vulnerability
3. Allow reasonable time for response before disclosure
4. We'll work with you to resolve the issue promptly

### Security Review Process

All security-related changes undergo additional review:

1. Security team review
2. Automated security testing
3. Manual penetration testing (for major changes)
4. Documentation of security implications

## 🎯 Development Guidelines

### Adding New Features

1. **Plan First**: Discuss large features in GitHub Discussions
2. **Start Small**: Break features into smaller, reviewable chunks
3. **Test Coverage**: Maintain or improve test coverage
4. **Documentation**: Update relevant documentation
5. **Backward Compatibility**: Avoid breaking changes when possible

### Code Organization

```
securestor/
├── cmd/                    # Main applications
│   ├── api/               # API server
│   └── migrate/           # Database migrations
├── internal/              # Private application code
│   ├── handlers/          # HTTP handlers
│   ├── models/            # Data models
│   ├── service/           # Business logic
│   └── repository/        # Data access layer
├── pkg/                   # Public library code
├── tests/                 # Additional test files
├── frontend/              # React frontend
└── docs/                  # Documentation
```

### Database Migrations

When adding database changes:

1. Create a new migration file: `migrations/XXX_description.sql`
2. Include both `UP` and `DOWN` migrations
3. Test migrations thoroughly
4. Update model definitions

```sql
-- migrations/025_add_new_table.sql

-- UP
CREATE TABLE new_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- DOWN
DROP TABLE IF EXISTS new_table;
```

## 🤝 Community

- **GitHub Discussions**: General discussions, Q&A, feature requests
- **GitHub Issues**: Bug reports and specific improvement requests
- **Discord**: Real-time chat (coming soon)
- **Twitter**: [@securestor](https://twitter.com/securestor) - Updates and announcements

## ⚖️ License

By contributing, you agree that your contributions will be licensed under the same AGPL-3.0 License that covers the project. Feel free to contact the maintainers if that's a concern.

## 👥 Recognition

Contributors are recognized in several ways:

- Listed in the project's README
- Mentioned in release notes for significant contributions
- Invited to join the contributors team for ongoing contributors

## 📞 Getting Help

If you need help with contributing:

1. Check existing documentation
2. Search GitHub Issues and Discussions
3. Create a new Discussion with the "help" label
4. Join our community chat (Discord link coming soon)

## 🎉 Thank You!

Your contributions make SecureStor better for everyone. We appreciate your time and effort in helping improve the project!

---

**Happy Contributing! 🚀**