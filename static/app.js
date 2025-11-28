let currentYear = 0;
let currentPage = 1;
let dp = null;

document.addEventListener('DOMContentLoaded', () => {
    initYearFilter();
    loadAnimeList();
    initSearch();
});

function initYearFilter() {
    const container = document.getElementById('yearFilter');
    let html = '<button class="year-btn active" onclick="filterYear(0)">全部</button>';
    for (let y = 2024; y >= 2000; y--) {
        html += `<button class="year-btn" onclick="filterYear(${y})">${y}</button>`;
    }
    container.innerHTML = html;
}

function filterYear(year) {
    currentYear = year;
    currentPage = 1;
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

    document.getElementById('fileList').innerHTML = '<h3>📂 选择视频源</h3><div class="loading">请在网盘中查找对应文件...</div>';
    modal.classList.add('show');
    
    // 尝试在网盘中搜索匹配的文件夹
    searchInStorage(anime.name_cn || anime.name);
}

async function searchInStorage(animeName) {
    const fileList = document.getElementById('fileList');
    // 这里可以扩展：遍历网盘目录匹配番剧名
    fileList.innerHTML = `
        <h3>📂 选择视频源</h3>
        <p style="color:#888;margin-bottom:15px">在网盘中查找: ${animeName}</p>
        <div class="file-item" onclick="browseStorage('/onedrive')">📁 OneDrive</div>
        <div class="file-item" onclick="browseStorage('/pikpak')">📁 PikPak</div>
    `;
}

async function browseStorage(path) {
    const fileList = document.getElementById('fileList');
    fileList.innerHTML = '<h3>📂 文件列表</h3><div class="loading">加载中...</div>';

    const resp = await fetch(`/api/list?path=${encodeURIComponent(path)}`);
    const result = await resp.json();
    const files = result.data?.content || [];

    let html = `<h3>📂 ${path}</h3>`;
    if (path !== '/') {
        const parent = path.split('/').slice(0, -1).join('/') || '/';
        html += `<div class="file-item" onclick="browseStorage('${parent}')">⬆️ 返回上级</div>`;
    }

    files.sort((a,b) => (b.is_dir - a.is_dir) || a.name.localeCompare(b.name));
    html += files.map(f => {
        const fullPath = path + '/' + f.name;
        if (f.is_dir) {
            return `<div class="file-item" onclick="browseStorage('${fullPath}')">📁 ${f.name}</div>`;
        }
        if (/\.(mp4|mkv|avi|webm)$/i.test(f.name)) {
            return `<div class="file-item" onclick="playVideo('${fullPath}')">🎬 ${f.name}</div>`;
        }
        return '';
    }).join('');

    fileList.innerHTML = html;
}

function closeModal() {
    document.getElementById('animeModal').classList.remove('show');
}

async function playVideo(path) {
    const container = document.getElementById('playerContainer');
    container.classList.add('show');
    closeModal();

    const resp = await fetch(`/api/get?path=${encodeURIComponent(path)}`);
    const result = await resp.json();

    if (result.code !== 200 || !result.data?.raw_url) {
        alert('获取播放地址失败');
        closePlayer();
        return;
    }

    if (dp) dp.destroy();
    dp = new DPlayer({
        container: document.getElementById('dplayer'),
        video: { url: result.data.raw_url, type: 'auto' },
        autoplay: true
    });
}

function closePlayer() {
    document.getElementById('playerContainer').classList.remove('show');
    if (dp) { dp.destroy(); dp = null; }
}

document.addEventListener('keydown', e => { if (e.key === 'Escape') { closePlayer(); closeModal(); } });
