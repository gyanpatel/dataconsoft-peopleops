export default {
    async fetch(request, env, ctx) {
      const url = new URL(request.url);
  
      // 1. API Proxy Pass (Replacing Nginx location /api/)
      if (url.pathname.startsWith('/api/')) {
        // Replace this with your public domain API endpoint or Cloudflare Tunnel address
        const API_UPSTREAM = 'http://144.21.56.192:8080'; 
        
        const newUrl = API_UPSTREAM + url.pathname + url.search;
        
        // Clone the request to modify headers, matching Nginx headers
        const modifiedRequest = new Request(newUrl, {
          method: request.method,
          headers: new Headers(request.headers),
          body: request.body,
          redirect: 'manual'
        });
        
        // Forward the request to your backend API
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
        // Fallback fallback to index.html if asset fetching throws an error
        const indexRequest = new Request(new URL('/index.html', request.url));
        return await env.ASSETS.fetch(indexRequest);
      }
    },
  };
  