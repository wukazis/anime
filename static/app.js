let currentYear = 0;
let currentPage = 1;
let dp = null;

// 视频源模式: 'clouddrive' 或 'openlist'
const VIDEO_MODE = 'clouddrive';

// CloudDrive2 配置
const CLOUDDRIVE_PATH = '/OneDrive';  // WebDAV 中的 OneDrive 挂载路径

// Cloudflare Worker 代理（openlist 模式时使用）
const CF_WORKER_URL = 'https://odd.wukazi.xyz';

document.addEventListener('DOMContentLoaded', () => {
    restoreStateFromURL();
    initYearFilter();
    loadAnimeList();
    initSearch();
});

function restoreStateFromURL() {
    const params = new URLSearchParams(window.location.search);
    currentYear = parseInt(params.get('year')) || 0;
    currentPage = parseInt(params.get('page')) || 1;
}

function updateURL() {
    const params = new URLSearchParams();
    if (currentYear) params.set('year', currentYear);
    if (currentPage > 1) params.set('page', currentPage);
    const newURL = params.toString() ? `?${params.toString()}` : window.location.pathname;
    history.replaceState(null, '', newURL);
}

function initYearFilter() {
    const container = document.getElementById('yearFilter');
    let html = `<button class="year-btn ${currentYear === 0 ? 'active' : ''}" onclick="filterYear(0)">全部</button>`;
    for (let y = 2025; y >= 2000; y--) {
        html += `<button class="year-btn ${currentYear === y ? 'active' : ''}" onclick="filterYear(${y})">${y}</button>`;
    }
    container.innerHTML = html;
}

function filterYear(year) {
    currentYear = year;
    currentPage = 1;
    updateURL();
    document.querySelectorAll('.year-btn').forEach(btn => btn.classList.remove('active'));
    event.target.classList.add('active');
    loadAnimeList();
}

async function loadAnimeList() {
    const grid = document.getElementById('animeGrid');
    grid.innerHTML = '<div class="loading">加载中...</div>';

    const url = `/api/anime?year=${currentYear}&page=${currentPage}`;
    const resp = await fetch(url);
    const result = await resp.json();

    if (!result.data || result.data.length === 0) {
        grid.innerHTML = '<div class="loading">暂无数据</div>';
        return;
    }

    grid.innerHTML = result.data.map(anime => `
        <div class="anime-card ${anime.has_resource ? 'has-resource' : 'no-resource'}" onclick="showAnimeDetail(${JSON.stringify(anime).replace(/"/g, '&quot;')})">
            <img src="${anime.cover || '/static/no-cover.png'}" alt="${anime.name_cn || anime.name}" loading="lazy">
            <div class="info">
                <div class="title">${anime.name_cn || anime.name}</div>
                <div class="meta">${anime.year} · <span class="score">${anime.score || '-'}</span></div>
            </div>
        </div>
    `).join('');

    renderPagination(result.total, result.page);
}

function renderPagination(total, page) {
    const totalPages = Math.ceil(total / 24);
    const container = document.getElementById('pagination');
    if (totalPages <= 1) { container.innerHTML = ''; return; }

    let html = `<button class="page-btn" onclick="goPage(${page-1})" ${page<=1?'disabled':''}>上一页</button>`;
    html += `<span style="color:#888;padding:8px">第 ${page}/${totalPages} 页</span>`;
    html += `<button class="page-btn" onclick="goPage(${page+1})" ${page>=totalPages?'disabled':''}>下一页</button>`;
    container.innerHTML = html;
}

function goPage(page) {
    currentPage = page;
    updateURL();
    loadAnimeList();
    window.scrollTo(0, 0);
}

function initSearch() {
    const input = document.getElementById('searchInput');
    const results = document.getElementById('searchResults');
    let timer;

    input.addEventListener('input', () => {
        clearTimeout(timer);
        const q = input.value.trim();
        if (!q) { results.classList.remove('show'); return; }
        timer = setTimeout(() => searchAnime(q), 300);
    });

    input.addEventListener('blur', () => setTimeout(() => results.classList.remove('show'), 200));
}

async function searchAnime(q) {
    const results = document.getElementById('searchResults');
    const resp = await fetch(`/api/anime/search?q=${encodeURIComponent(q)}`);
    const data = await resp.json();

    if (!data || data.length === 0) {
        results.innerHTML = '<div class="search-item">无结果</div>';
    } else {
        results.innerHTML = data.map(a => `
            <div class="search-item" onclick="showAnimeDetail(${JSON.stringify(a).replace(/"/g, '&quot;')})">${a.name_cn || a.name} (${a.year})</div>
        `).join('');
    }
    results.classList.add('show');
}

function showAnimeDetail(anime) {
    const modal = document.getElementById('animeModal');
    const detail = document.getElementById('animeDetail');

    detail.innerHTML = `
        <img src="${anime.cover || '/static/no-cover.png'}" alt="">
        <div class="detail-info">
            <h2>${anime.name_cn || anime.name}</h2>
            <p>${anime.name}</p>
            <p>📅 ${anime.date || anime.year} · ⭐ ${anime.score || '-'}</p>
            <div class="tags">${(anime.tags||[]).map(t => `<span class="tag">${t}</span>`).join('')}</div>
            <p class="summary">${anime.summary || '暂无简介'}</p>
        </div>
    `;

    const fileList = document.getElementById('fileList');

    if (anime.has_resource) {
        fileList.innerHTML = '<h3>🎬 选集</h3><div class="loading">加载中...</div>';
        modal.classList.add('show');
        loadEpisodes(anime.name_cn || anime.name, anime.year);
    } else {
        fileList.innerHTML = '<h3>🎬 选集</h3><p style="color:#888">暂无资源</p>';
        modal.classList.add('show');
    }
}

async function loadEpisodes(name, year) {
    const fileList = document.getElementById('fileList');
    try {
        const resp = await fetch(`/api/anime/episodes?name=${encodeURIComponent(name)}&year=${year}`);
        const data = await resp.json();

        if (!data.episodes || data.episodes.length === 0) {
            fileList.innerHTML = '<h3>🎬 选集</h3><p style="color:#888">暂无视频文件</p>';
            return;
        }

        let html = `<h3>🎬 选集 (${data.episodes.length}集)</h3><div class="episode-grid">`;
        data.episodes.forEach((ep, idx) => {
            const num = idx + 1;
            html += `<div class="episode-btn" onclick="playVideo('${ep.path.replace(/'/g, "\\'")}', '${ep.name.replace(/'/g, "\\'")}')" title="${ep.name}">${num}</div>`;
        });
        html += '</div>';
        fileList.innerHTML = html;
    } catch (e) {
        fileList.innerHTML = '<h3>🎬 选集</h3><p style="color:#f66">加载失败</p>';
    }
}

function closeModal() {
    document.getElementById('animeModal').classList.remove('show');
}

async function playVideo(path, filename) {
    const container = document.getElementById('playerContainer');
    container.classList.add('show');
    closeModal();

    let videoUrl;

    if (VIDEO_MODE === 'clouddrive') {
        // CloudDrive2 模式：直接走后端流式代理
        // 路径格式: /onedrive/anime/xxx/file.mp4 -> /api/stream/OneDrive/anime/xxx/file.mp4
        const streamPath = path.replace(/^\/onedrive/i, CLOUDDRIVE_PATH);
        videoUrl = `/api/stream${streamPath}`;
    } else {
        // OpenList 模式：获取直链
        const resp = await fetch(`/api/get?path=${encodeURIComponent(path)}`);
        const result = await resp.json();

        if (result.code !== 200 || !result.data?.raw_url) {
            alert('获取播放地址失败');
            closePlayer();
            return;
        }

        videoUrl = result.data.raw_url;

        // CF Worker 代理
        if (CF_WORKER_URL && isOneDriveUrl(videoUrl)) {
            videoUrl = `${CF_WORKER_URL}/?url=${encodeURIComponent(videoUrl)}`;
        }
    }

    if (dp) dp.destroy();
    dp = new DPlayer({
        container: document.getElementById('dplayer'),
        video: { url: videoUrl, type: 'auto' },
        autoplay: true
    });
}

function isOneDriveUrl(url) {
    const hosts = ['sharepoint.com', 'onedrive.live.com', '1drv.ms', 'storage.live.com'];
    try {
        const hostname = new URL(url).hostname;
        return hosts.some(h => hostname.endsWith(h));
    } catch {
        return false;
    }
}

function closePlayer() {
    document.getElementById('playerContainer').classList.remove('show');
    if (dp) { dp.destroy(); dp = null; }
}

document.addEventListener('keydown', e => { if (e.key === 'Escape') { closePlayer(); closeModal(); } });
