/**
 * Multi-Tenant Proxy Configuration for React Dev Server
 * Handles subdomain-based routing in development
 * 
 * This proxy ensures that API calls from:
 * - alpha.localhost:3000 → alpha.localhost (backend port 80)
 * - beta.localhost:3000 → beta.localhost (backend port 80)
 * - localhost:3000 → localhost (backend port 80)
 */

const { createProxyMiddleware } = require('http-proxy-middleware');

module.exports = function(app) {
  // Proxy for /api/* endpoints
  app.use(
    '/api',
    createProxyMiddleware({
      target: 'http://localhost',
      changeOrigin: false, // Keep original host header (preserves subdomain)
      secure: false,
      logLevel: 'debug',
      onProxyReq: (proxyReq, req, res) => {
        // Preserve the original host with subdomain
        const originalHost = req.headers.host;
        
        // Extract subdomain from original host
        const hostname = originalHost.split(':')[0];
        
        // Set target to preserve subdomain
        // alpha.localhost:3000 → alpha.localhost
        // beta.localhost:3000 → beta.localhost
        proxyReq.setHeader('Host', hostname);
        
      },
      onError: (err, req, res) => {
        console.error('[Proxy Error]', err.message);
        res.writeHead(502, {
          'Content-Type': 'application/json',
        });
        res.end(JSON.stringify({
          error: 'Proxy Error',
          message: err.message,
          hint: 'Make sure backend is running on port 80',
        }));
      },
      router: (req) => {
        // Dynamic target based on the request host
        const host = req.headers.host.split(':')[0]; // Remove port
        const target = `http://${host}`;
        return target;
      },
    })
  );
};
