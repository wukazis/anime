# Cloudflare Worker - OneDrive 代理

用于加速 OneDrive 直链访问，解决国内访问慢的问题。

## 部署步骤

### 1. 登录 Cloudflare Dashboard
访问 https://dash.cloudflare.com/

### 2. 创建 Worker
1. 左侧菜单选择 **Workers & Pages**
2. 点击 **Create application** → **Create Worker**
3. 给 Worker 起个名字，如 `onedrive-proxy`
4. 点击 **Deploy**

### 3. 编辑代码
1. 点击 **Edit code**
2. 删除默认代码，粘贴 `onedrive-proxy.js` 的内容
3. 点击 **Save and Deploy**

### 4. 获取 Worker URL
部署后会得到一个 URL，格式如：
```
https://onedrive-proxy.your-name.workers.dev
```

### 5. 配置前端
编辑 `static/app.js`，填入你的 Worker URL：
```javascript
const CF_WORKER_URL = 'https://onedrive-proxy.your-name.workers.dev';
```

## 使用说明

配置完成后，播放视频时会自动：
1. 检测是否是 OneDrive/SharePoint 链接
2. 如果是，通过 CF Worker 代理访问
3. 利用 Cloudflare 全球 CDN 加速

## 限制

- Cloudflare Workers 免费版每天 10 万次请求
- 单次请求最大 100MB（视频会分段请求，一般没问题）
- 如果流量大，考虑升级付费版

## 测试

直接访问：
```
https://your-worker.workers.dev/?url=<onedrive直链>
```
