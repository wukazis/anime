let currentPath = '/';
let dp = null;

document.addEventListener('DOMContentLoaded', () => {
    loadDirectory('/');
});

async function loadDirectory(path) {
    currentPath = path;
    updateBreadcrumb();
    
    const fileList = document.getElementById('fileList');
    fileList.innerHTML = '<div class="loading">加载中...</div>';

    try {
        const resp = await fetch(`/api/list?path=${encodeURIComponent(path)}`);
        const result = await resp.json();
        
        if (result.code !== 200) {
            fileList.innerHTML = `<div class="loading">错误: ${result.message}</div>`;
            return;
        }

        const files = result.data.content || [];
        if (files.length === 0) {
            fileList.innerHTML = '<div class="loading">空目录</div>';
            return;
        }

        // 排序：文件夹在前
        files.sort((a, b) => {
            if (a.is_dir && !b.is_dir) return -1;
            if (!a.is_dir && b.is_dir) return 1;
            return a.name.localeCompare(b.name);
        });

        fileList.innerHTML = files.map(file => `
            <div class="file-item" onclick="${file.is_dir ? `loadDirectory('${path === '/' ? '' : path}/${file.name}')` : `playVideo('${path === '/' ? '' : path}/${file.name}')`}">
                <div class="icon">${getIcon(file)}</div>
                <div class="name">${file.name}</div>
            </div>
        `).join('');
    } catch (err) {
        fileList.innerHTML = `<div class="loading">请求失败: ${err.message}</div>`;
    }
}

function getIcon(file) {
    if (file.is_dir) return '📁';
    const ext = file.name.split('.').pop().toLowerCase();
    if (['mp4', 'mkv', 'avi', 'webm', 'mov'].includes(ext)) return '🎬';
    if (['mp3', 'flac', 'wav', 'aac'].includes(ext)) return '🎵';
    if (['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext)) return '🖼️';
    return '📄';
}

function updateBreadcrumb() {
    const breadcrumb = document.getElementById('breadcrumb');
    const parts = currentPath.split('/').filter(p => p);
    
    let html = '<span onclick="loadDirectory(\'/\')">根目录</span>';
    let path = '';
    
    for (const part of parts) {
        path += '/' + part;
        html += ` / <span onclick="loadDirectory('${path}')">${part}</span>`;
    }
    
    breadcrumb.innerHTML = html;
}

async function playVideo(path) {
    const container = document.getElementById('playerContainer');
    container.style.display = 'flex';
    
    try {
        const resp = await fetch(`/api/get?path=${encodeURIComponent(path)}`);
        const result = await resp.json();
        
        if (result.code !== 200 || !result.data.raw_url) {
            alert('获取播放地址失败: ' + result.message);
            closePlayer();
            return;
        }

        if (dp) {
            dp.destroy();
        }

        dp = new DPlayer({
            container: document.getElementById('dplayer'),
            video: {
                url: result.data.raw_url,
                type: 'auto'
            },
            autoplay: true
        });
    } catch (err) {
        alert('播放失败: ' + err.message);
        closePlayer();
    }
}

function closePlayer() {
    const container = document.getElementById('playerContainer');
    container.style.display = 'none';
    if (dp) {
        dp.destroy();
        dp = null;
    }
}

// ESC 关闭播放器
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closePlayer();
});
