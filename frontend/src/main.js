import './style.css';

import { GetRecentFolders, Search, IsDirectory, Open } from '../wailsjs/go/main/App';
import { EventsOn, WindowHide } from '../wailsjs/runtime/runtime';

const input = document.getElementById('search');
const empty = document.getElementById('empty');
const emptyTitle = document.getElementById('empty-title');
const emptyHint = document.getElementById('empty-hint');
const results = document.getElementById('results');

// Panel state: the full list (newest first), the filtered list, and the
// selected index within the filtered list (-1 = nothing selected).
let all = [];
let filtered = [];
let selected = -1;

// The panel opens (hotkey or tray): clear the box, focus it and reload the
// list so it reflects anything recorded since the last open.
EventsOn('panel-opened', () => {
    input.value = '';
    input.focus();
    load();
});

// Load the full Recent Folders list and reset the filter to show everything.
async function load() {
    try {
        all = (await GetRecentFolders()) || [];
    } catch (err) {
        console.error('GetRecentFolders failed:', err);
        all = [];
    }
    filtered = all;
    selected = all.length ? 0 : -1;
    render();
}

// Keyboard navigation is bound to the window: the search box is focused on
// open and is the only focusable element, so everything stays reachable
// without the mouse even if focus strays.
window.addEventListener('keydown', (e) => {
    switch (e.key) {
        case 'ArrowDown':
            e.preventDefault();
            moveSelected(1);
            break;
        case 'ArrowUp':
            e.preventDefault();
            moveSelected(-1);
            break;
        case 'Enter':
            e.preventDefault();
            openTarget();
            break;
        case 'Escape':
            WindowHide();
            break;
    }
});

// Each keystroke filters the list instantly through the Go fuzzy search.
input.addEventListener('input', () => {
    Search(input.value.trim())
        .then((list) => {
            filtered = list || [];
            selected = filtered.length ? 0 : -1;
            render();
        })
        .catch((err) => console.error('Search failed:', err));
});

// Clicking an entry opens it (mouse is optional but supported).
results.addEventListener('click', (e) => {
    const li = e.target.closest('li');
    if (!li) return;
    const folder = filtered[Number(li.dataset.index)];
    if (folder) doOpen(folder.path);
});

function moveSelected(delta) {
    if (!filtered.length) return;
    selected = Math.min(filtered.length - 1, Math.max(0, selected + delta));
    render();
}

// Decide what Enter opens: if the search box holds a real directory path,
// open that directly (bootstrap any folder — spec story 5, works even when
// history is empty); otherwise open the selected entry.
async function openTarget() {
    const typed = cleanPath(input.value.trim());
    if (typed && (await IsDirectory(typed))) {
        doOpen(typed);
        return;
    }
    if (selected >= 0 && filtered[selected]) {
        doOpen(filtered[selected].path);
    }
}

// Open the folder, refresh so it is bumped to the top, then dismiss the
// panel. A failed open keeps the panel open and surfaces the reason.
async function doOpen(path) {
    try {
        await Open(path);
    } catch (err) {
        console.error('Open failed:', err);
        showEmpty('无法打开该文件夹', String(err));
        return;
    }
    await load();
    WindowHide();
}

// cleanPath strips surrounding quotes that often come with a pasted path.
function cleanPath(p) {
    if (p.length >= 2 && p[0] === '"' && p[p.length - 1] === '"') {
        return p.slice(1, -1);
    }
    return p;
}

function render() {
    results.innerHTML = '';
    if (!all.length) {
        // First run / empty history: show the bootstrap path.
        showEmpty('还没有历史记录', '粘贴或输入文件夹路径，按回车直接打开');
        return;
    }
    if (!filtered.length) {
        // History exists but nothing matched the query.
        showEmpty('没有匹配的文件夹', '输入完整路径，按回车可直接打开');
        return;
    }
    empty.classList.add('hidden');
    const frag = document.createDocumentFragment();
    filtered.forEach((folder, i) => {
        const li = document.createElement('li');
        li.dataset.index = String(i);
        li.textContent = folder.name || folder.path;
        li.title = folder.path;
        if (i === selected) li.classList.add('selected');
        frag.appendChild(li);
    });
    results.appendChild(frag);
    scrollSelectedIntoView();
}

function showEmpty(title, hint) {
    results.innerHTML = '';
    emptyTitle.textContent = title;
    emptyHint.textContent = hint;
    empty.classList.remove('hidden');
}

function scrollSelectedIntoView() {
    if (selected < 0) return;
    const el = results.children[selected];
    if (el) el.scrollIntoView({ block: 'nearest' });
}

// Initial paint in case the webview loads before the first panel-opened.
load();
