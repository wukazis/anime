/**
 * Cloudflare Worker - OneDrive 直链代理
 * 部署到 Cloudflare Workers 后，可以加速 OneDrive 文件访问
 * 
 * 使用方法：
 * 原始 URL: https://xxx.sharepoint.com/xxx/file.mp4
 * 代理 URL: https://your-worker.workers.dev/?url=https://xxx.sharepoint.com/xxx/file.mp4
 */

export default {
  async fetch(request) {
    const url = new URL(request.url);
    
    // 获取要代理的 URL
    const targetUrl = url.searchParams.get('url');
    
    if (!targetUrl) {
      return new Response(JSON.stringify({
        error: 'Missing url parameter',
        usage: 'https://your-worker.workers.dev/?url=<onedrive-direct-link>'
      }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // 验证是否是 OneDrive/SharePoint URL
    const allowedHosts = [
      'sharepoint.com',
      'onedrive.live.com',
      '1drv.ms',
      'storage.live.com'
    ];
    
    try {
      const targetHost = new URL(targetUrl).hostname;
      const isAllowed = allowedHosts.some(h => targetHost.endsWith(h));
      
      if (!isAllowed) {
        return new Response(JSON.stringify({
          error: 'Only OneDrive/SharePoint URLs are allowed'
        }), {
          status: 403,
          headers: { 'Content-Type': 'application/json' }
        });
      }
    } catch (e) {
      return new Response(JSON.stringify({ error: 'Invalid URL' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    // 构建代理请求
    const headers = new Headers();
    
    // 转发 Range 头（支持视频拖动）
    if (request.headers.has('Range')) {
      headers.set('Range', request.headers.get('Range'));
    }
    
    // 转发其他有用的头
    const forwardHeaders = ['Accept', 'Accept-Encoding', 'Accept-Language'];
    forwardHeaders.forEach(h => {
      if (request.headers.has(h)) {
        headers.set(h, request.headers.get(h));
      }
    });

    try {
      const response = await fetch(targetUrl, {
        method: request.method,
        headers: headers,
        redirect: 'follow'
      });

      // 构建响应头
      const responseHeaders = new Headers();
      
      // 复制重要的响应头
      const copyHeaders = [
        'Content-Type',
        'Content-Length',
        'Content-Range',
        'Accept-Ranges',
        'ETag',
        'Last-Modified'
      ];
      
      copyHeaders.forEach(h => {
        if (response.headers.has(h)) {
          responseHeaders.set(h, response.headers.get(h));
        }
      });

      // CORS 头
      responseHeaders.set('Access-Control-Allow-Origin', '*');
      responseHeaders.set('Access-Control-Allow-Methods', 'GET, HEAD, OPTIONS');
      responseHeaders.set('Access-Control-Allow-Headers', 'Range');
      responseHeaders.set('Access-Control-Expose-Headers', 'Content-Length, Content-Range');
      
      // 缓存控制
      responseHeaders.set('Cache-Control', 'public, max-age=86400');

      // OPTIONS 预检请求
      if (request.method === 'OPTIONS') {
        return new Response(null, { status: 204, headers: responseHeaders });
      }

      return new Response(response.body, {
        status: response.status,
        headers: responseHeaders
      });

    } catch (e) {
      return new Response(JSON.stringify({ error: e.message }), {
        status: 500,
        headers: { 'Content-Type': 'application/json' }
      });
    }
  }
};
