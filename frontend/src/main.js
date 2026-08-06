import './style.css';

import { GetRecentFolders, Search, IsDirectory, Open, GetSoftwareList, OpenWith } from '../wailsjs/go/main/App';
import { EventsOn, WindowHide } from '../wailsjs/runtime/runtime';

const input = document.getElementById('search');
const empty = document.getElementById('empty');
const emptyTitle = document.getElementById('empty-title');
const emptyHint = document.getElementById('empty-hint');
const results = document.getElementById('results');

// Panel state.
//   all       — full Recent Folders list (newest first)
//   filtered  — current fuzzy-filtered subset
//   selected  — index within filtered (-1 = nothing selected)
//   software  — preset Software List; its index + 1 is the panel key (1-9)
//   visible   — whether the panel is user-visible; all keyboard input is
//               ignored while false. The webview can keep processing keys
//               briefly after the host window is hidden, so this guard stops
//               a stray keystroke from opening a folder once dismissed.
let all = [];
let filtered = [];
let selected = -1;
let software = [];
let visible = false;

// The panel opens (hotkey or tray): clear the box, focus it and reload both
// the folder list and the software list so they reflect current state.
EventsOn('panel-opened', () => {
    visible = true;
    input.value = '';
    input.focus();
    load();
});

// The panel was hidden from the Go side (deactivation, or the hotkey toggled
// it closed). Disarm keyboard input and drop focus so no stray keystroke can
// trigger an action while the window is dismissed.
EventsOn('panel-hidden', () => {
    dismiss();
});

// Load Recent Folders and the Software List together, then reset the filter.
async function load() {
    try {
        [all, software] = await Promise.all([GetRecentFolders(), GetSoftwareList()]);
    } catch (err) {
        console.error('load failed:', err);
        all = [];
        software = [];
    }
    all = all || [];
    software = software || [];
    filtered = all;
    selected = all.length ? 0 : -1;
    render();
}

// Keyboard navigation is bound to the window: the search box is focused on
// open and is the only focusable element, so everything stays reachable
// without the mouse even if focus strays. Input is ignored entirely while
// the panel is dismissed.
window.addEventListener('keydown', (e) => {
    if (!visible) {
        return;
    }
    // Digit keys 1-9 trigger the Software List action of that ordinal on the
    // resolved target folder (typed path wins, else the selected entry).
    if (e.key >= '1' && e.key <= '9') {
        e.preventDefault();
        openWithKey(Number(e.key) - 1);
        return;
    }
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
            openDefault();
            break;
        case 'Escape':
            closePanel();
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

// Click an entry: a software badge opens it with that app; anywhere else on
// the row opens it with the default Explorer action.
results.addEventListener('click', (e) => {
    const li = e.target.closest('li');
    if (!li) return;
    const folder = filtered[Number(li.dataset.index)];
    if (!folder) return;
    const badge = e.target.closest('[data-sw]');
    if (badge !== null) {
        runAndRefresh(() => OpenWith(folder.path, Number(badge.dataset.sw)));
    } else {
        runAndRefresh(() => Open(folder.path));
    }
});

function moveSelected(delta) {
    if (!filtered.length) return;
    selected = Math.min(filtered.length - 1, Math.max(0, selected + delta));
    render();
}

// resolveTarget picks the folder an action acts on: if the search box holds a
// real directory path, use that (bootstrap any folder — spec story 5, works
// even with empty history); otherwise use the selected entry.
async function resolveTarget() {
    const typed = cleanPath(input.value.trim());
    if (typed && (await IsDirectory(typed))) return typed;
    if (selected >= 0 && filtered[selected]) return filtered[selected].path;
    return null;
}

// Enter: open the resolved target with the default Explorer action.
async function openDefault() {
    const path = await resolveTarget();
    if (path) runAndRefresh(() => Open(path));
}

// A digit key: open the resolved target with software[index], if that many
// apps are configured. Keys beyond the list are ignored.
async function openWithKey(index) {
    if (index >= software.length) return;
    const path = await resolveTarget();
    if (path) runAndRefresh(() => OpenWith(path, index));
}

// Run an open action, then refresh (so the opened folder is bumped to the
// top) and dismiss the panel. A failed open keeps the panel open and
// surfaces the reason as an empty-state message.
async function runAndRefresh(action) {
    try {
        await action();
    } catch (err) {
        console.error('open failed:', err);
        showEmpty('无法打开该文件夹', String(err));
        return;
    }
    await load();
    closePanel();
}

// closePanel hides the window from the JS side (Escape, or after a successful
// open) and immediately disarms input locally, so there is no window between
// the hide and the Go-side «panel-hidden» event in which a key could leak.
function closePanel() {
    WindowHide();
    dismiss();
}

// dismiss marks the panel as not user-visible, drops keyboard focus from the
// search box and clears the selection, so the webview cannot act on input
// while the host window is hidden.
function dismiss() {
    visible = false;
    selected = -1;
    if (document.activeElement && document.activeElement.blur) {
        document.activeElement.blur();
    }
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
        // First run / empty history: bootstrap path. Mention the app keys
        // only when at least one is configured.
        const hint = software.length
            ? '粘贴或输入文件夹路径，回车用资源管理器打开（或按 1-9 用应用打开）'
            : '粘贴或输入文件夹路径，按回车直接打开';
        showEmpty('还没有历史记录', hint);
        return;
    }
    if (!filtered.length) {
        showEmpty('没有匹配的文件夹', '输入完整路径，按回车可直接打开');
        return;
    }
    empty.classList.add('hidden');
    const frag = document.createDocumentFragment();
    filtered.forEach((folder, i) => {
        const li = document.createElement('li');
        li.dataset.index = String(i);
        li.title = folder.path;

        const name = document.createElement('span');
        name.className = 'name';
        name.textContent = folder.name || folder.path;
        li.appendChild(name);

        if (software.length) {
            const actions = document.createElement('span');
            actions.className = 'actions';
            software.forEach((sw, si) => {
                const badge = document.createElement('span');
                badge.className = 'sw';
                badge.dataset.sw = String(si);
                if (si <= 8) {
                    const k = document.createElement('kbd');
                    k.textContent = String(si + 1);
                    badge.appendChild(k);
                }
                badge.appendChild(document.createTextNode(sw.name));
                actions.appendChild(badge);
            });
            li.appendChild(actions);
        }

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
