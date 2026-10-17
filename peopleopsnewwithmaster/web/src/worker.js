export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
 
    // 1. API Proxy Pass (Replacing Nginx location /api/)
    if (url.pathname.startsWith('/api/')) {
      const API_UPSTREAM = 'http://144.21.56.192:8080'; 
      const newUrl = API_UPSTREAM + url.pathname + url.search;
      
      // Clone the incoming headers so we can safely modify them
      const modifiedHeaders = new Headers(request.headers);
      
      // CRITICAL FIX: Update the Host header to match the backend destination
      // This prevents the backend server from throwing a 403 Forbidden
      modifiedHeaders.set('Host', '144.21.56.192:8080');

      // Forward standard Nginx proxy headers for tracking real user IPs
      modifiedHeaders.set('X-Forwarded-Host', url.host);
      modifiedHeaders.set('X-Forwarded-Proto', url.protocol.replace(':', ''));

      // Safely manage the request body (important for POST/PUT login payloads)
      let requestBody = null;
      if (request.method !== 'GET' && request.method !== 'HEAD') {
        // Use clone() to prevent stream consumption errors
        requestBody = request.clone().body;
      }
      
      const modifiedRequest = new Request(newUrl, {
        method: request.method,
        headers: modifiedHeaders,
        body: requestBody,
        redirect: 'manual'
      });
      
      // Forward the request to your public backend API IP
      return await fetch(modifiedRequest);
    }

    // 2. Serve Static Assets & Handle SPA Routing (Replacing Nginx try_files)
    try {
      const assetResponse = await env.ASSETS.fetch(request);
      
      // If the asset doesn't exist (404), fallback to index.html for SPA routing
      if (assetResponse.status === 404) {
        const indexRequest = new Request(new URL('/index.html', request.url));
        return await env.ASSETS.fetch(indexRequest);
      }
      
      return assetResponse;
    } catch (e) {
      // Fallback to index.html if asset fetching throws an error
      const indexRequest = new Request(new URL('/index.html', request.url));
      return await env.ASSETS.fetch(indexRequest);
    }
  },
};
