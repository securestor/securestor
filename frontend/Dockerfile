# Multi-stage build for optimized production frontend

# Stage 1: Build the React application
FROM node:18-alpine AS builder

WORKDIR /app

# Accept build argument for edition
ARG REACT_APP_EDITION=community
ENV REACT_APP_EDITION=${REACT_APP_EDITION}

# Accept build arguments for tenant configuration
ARG REACT_APP_DEFAULT_TENANT=admin
ARG REACT_APP_DEV_MODE=true
ARG REACT_APP_BASE_DOMAIN=localhost
ARG REACT_APP_API_URL=http://localhost:8080
ARG REACT_APP_BACKEND_PORT=8080
ENV REACT_APP_DEFAULT_TENANT=${REACT_APP_DEFAULT_TENANT}
ENV REACT_APP_DEV_MODE=${REACT_APP_DEV_MODE}
ENV REACT_APP_BASE_DOMAIN=${REACT_APP_BASE_DOMAIN}
ENV REACT_APP_API_URL=${REACT_APP_API_URL}
ENV REACT_APP_BACKEND_PORT=${REACT_APP_BACKEND_PORT}

# Copy package files
COPY package*.json ./

# Install dependencies with legacy peer deps to handle TypeScript version conflicts
RUN npm ci --legacy-peer-deps

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Stage 2: Serve with nginx
FROM nginx:alpine

# Copy custom nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built assets from builder stage
COPY --from=builder /app/build /usr/share/nginx/html

# Add healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:3000/ || exit 1

# Expose port
EXPOSE 3000

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
