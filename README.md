# Anime Site

基于 OpenList/AList 的动漫视频站，支持浏览网盘目录并在线播放。

## 功能

- 浏览 OpenList 挂载的网盘目录
- DPlayer 在线播放视频
- 面包屑导航
- 支持 PikPak、OneDrive 等网盘

## 运行

```bash
# 复制配置文件
cp config.example.json config.json

# 修改 config.json 中的 openlist_url

# 运行
go run main.go
```

访问 http://localhost:8888

## 配置

```json
{
  "port": "8888",
  "openlist_url": "https://your-openlist-url.com"
}
```
