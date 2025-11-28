// PikPak 批量离线下载脚本
// 使用方法：在 https://mypikpak.com 登录后，F12 打开控制台，粘贴运行

const magnets = [
// 从 matched_magnets.txt 复制磁力链接到这里，格式如下：
// { name: "犬夜叉 (2000)", url: "magnet:?xt=urn:btih:..." },
];

// 或者直接粘贴磁力链接数组
const magnetUrls = [
// "magnet:?xt=urn:btih:...",
// "magnet:?xt=urn:btih:...",
];

async function addOfflineDownload(url, name = '') {
    const token = localStorage.getItem('access_token');
    if (!token) {
        console.error('未找到 access_token，请先登录');
        return { error: 'not_logged_in' };
    }
    
    try {
        const resp = await fetch('https://api-drive.mypikpak.com/drive/v1/files', {
            method: 'POST',
            headers: {
                'Authorization': 'Bearer ' + token,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                kind: 'drive#file',
                upload_type: 'UPLOAD_TYPE_URL',
                url: { url },
                parent_id: '',  // 空为根目录，可以改成文件夹ID
                folder_type: ''
            })
        });
        return await resp.json();
    } catch (e) {
        return { error: e.message };
    }
}

async function batchDownload() {
    const tasks = magnets.length > 0 ? magnets : magnetUrls.map((url, i) => ({ name: `任务${i+1}`, url }));
    
    if (tasks.length === 0) {
        console.log('请先添加磁力链接到 magnets 或 magnetUrls 数组');
        return;
    }
    
    console.log(`开始批量添加 ${tasks.length} 个任务...`);
    
    const failed = [];
    let success = 0;
    
    for (let i = 0; i < tasks.length; i++) {
        const task = tasks[i];
        const url = task.url || task;
        const name = task.name || `任务${i+1}`;
        
        console.log(`[${i+1}/${tasks.length}] ${name}`);
        
        const result = await addOfflineDownload(url, name);
        
        if (result.file || result.task) {
            success++;
            console.log(`  ✅ 成功`);
        } else {
            failed.push({ name, url, error: result.error || JSON.stringify(result) });
            console.log(`  ❌ 失败: ${result.error || result.error_description || '未知错误'}`);
        }
        
        // 间隔 1 秒，避免请求过快
        await new Promise(r => setTimeout(r, 1000));
    }
    
    console.log(`\n完成！成功: ${success}, 失败: ${failed.length}`);
    
    if (failed.length > 0) {
        console.log('\n失败列表:');
        failed.forEach(f => console.log(`${f.name}: ${f.error}`));
        
        // 导出失败列表
        const failedText = failed.map(f => `# ${f.name}\n# 原因: ${f.error}\n${f.url}`).join('\n\n');
        console.log('\n失败列表文本（可复制保存）:');
        console.log(failedText);
    }
}

// 运行
batchDownload();
