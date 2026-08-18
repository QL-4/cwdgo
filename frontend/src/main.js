import './style.css';

import { GetRecentFolders, Search, IsDirectory, Open, HideAfterOpen, Record, GetSoftwareList, OpenWith, GetSettings, SaveSettings, AddSoftware, UpdateSoftware, DeleteSoftware } from '../wailsjs/go/main/App';
import { EventsOn, WindowHide } from '../wailsjs/runtime/runtime';

const panel = document.getElementById('panel');
const settingsView = document.getElementById('settings');
const input = document.getElementById('search');
const empty = document.getElementById('empty');
const emptyTitle = document.getElementById('empty-title');
const emptyHint = document.getElementById('empty-hint');
const results = document.getElementById('results');

// Settings view controls.
const backBtn = document.getElementById('back');
const historyLimit = document.getElementById('history-limit');
const autostart = document.getElementById('autostart');
const settingsMsg = document.getElementById('settings-msg');

// Software list controls.
const swList = document.getElementById('sw-list');
const swName = document.getElementById('sw-name');
const swExe = document.getElementById('sw-exe');
const swArgs = document.getElementById('sw-args');
const swForm = document.querySelector('.sw-form');
const swAddBtn = document.getElementById('sw-add');
const swNewBtn = document.getElementById('sw-new');
const swCancelBtn = document.getElementById('sw-cancel');
// editIndex is the index of the entry currently being edited, or -1 when
// the form is in «add» mode (or hidden).
let editIndex = -1;

// Panel state.
//   all       — full Recent Folders list (newest first)
//   filtered  — current fuzzy-filtered subset
//   selected  — index within filtered (-1 = nothing selected)
//   software  — preset Software List; its index + 1 is the panel key (1-9)
//   visible   — whether the window is user-visible; all keyboard input is
//               ignored while false. The webview can keep processing keys
//               briefly after the host window is hidden, so this guard stops
//               a stray keystroke from acting once dismissed.
//   view      — which view is showing ('panel' | 'settings'). Panel-only
//               shortcuts (digits, arrows, Enter) are gated on 'panel'; Esc
//               closes from either view.
let all = [];
let filtered = [];
let selected = -1;
let software = [];
let visible = false;
let view = 'panel';
// activeAction is the currently armed open method: -1 = default Explorer,
// 0..software.length-1 = the software entry to open with. Left/Right arrow
// keys cycle it; Enter opens the resolved folder with it.
let activeAction = -1;

function isSSHPath(path) {
    return /^[^:\\/]{2,}:\//.test(path || '');
}

// SSH projects can only be opened by Trae CN. Their original action index
// stays intact, so key 3 remains Trae CN while 1 and 2 are disabled.
function availableActionIndexes(folderOrPath) {
    const path = typeof folderOrPath === 'string' ? folderOrPath : folderOrPath?.path;
    return software
        .map((sw, index) => ({ sw, index }))
        .filter(({ sw }) => !isSSHPath(path) || sw.name.toLowerCase() === 'trae cn')
        .map(({ index }) => index);
}

function cycleAction(delta) {
    if (selected < 0 || !filtered[selected]) return;
    const choices = [-1, ...availableActionIndexes(filtered[selected])];
    let position = choices.indexOf(activeAction);
    if (position < 0) position = 0;
    activeAction = choices[(position + delta + choices.length) % choices.length];
    render();
}

// The panel opens (hotkey or tray): switch to the panel view, clear the
// box, focus it and reload both the folder list and the software list so
// they reflect current state.
EventsOn('panel-opened', () => {
    visible = true;
    showView('panel');
    input.value = '';
    input.focus();
    load();
});

// The settings view opens (tray «设置»). Switch to it and load the current
// persisted values into the form.
EventsOn('settings-opened', () => {
    visible = true;
    showView('settings');
    loadSettings();
});

// The window was hidden from the Go side (deactivation, or the hotkey
// toggled it closed). Disarm keyboard input and drop focus so no stray
// keystroke can trigger an action while the window is dismissed.
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
    // While Alt is held the OS is in menu mode: arrow keys would open the
    // window system menu, so ignore all panel shortcuts (the Go side also
    // clears that mode after the panel activates).
    if (e.altKey) {
        return;
    }
    // Esc closes the window from either view.
    if (e.key === 'Escape') {
        closePanel();
        return;
    }
    // Panel-only shortcuts are ignored while the settings view is up.
    if (view !== 'panel') {
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
        case 'ArrowRight':
            if (input.value.trim() !== '') return;
            e.preventDefault();
            cycleAction(1);
            break;
        case 'ArrowLeft':
            if (input.value.trim() !== '') return;
            e.preventDefault();
            cycleAction(-1);
            break;
        case 'Enter':
            e.preventDefault();
            openDefault();
            break;
        case 'Tab':
            // While searching: complete the search box with the selected
            // folder's full path, then re-filter so the list shows that
            // exact target. Pressing Enter afterwards records/opens it.
            e.preventDefault();
            if (input.value.trim() !== '' && filtered.length && selected >= 0) {
                input.value = filtered[selected].path;
                Search(input.value.trim())
                    .then((list) => {
                        filtered = list || [];
                        selected = filtered.length ? 0 : -1;
                        render();
                    })
                    .catch((err) => console.error('Search failed:', err));
            }
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
    const allowed = availableActionIndexes(filtered[selected]);
    if (activeAction >= 0 && !allowed.includes(activeAction)) activeAction = -1;
    render();
}

// resolveTarget picks the folder an action acts on: the SELECTED entry wins
// (its full path), so typing a query and pressing Enter records/opens the
// highlighted result; when nothing is selected (e.g. the typed path matched
// nothing on disk), fall back to the typed text if it is a real directory —
// this keeps the bootstrap path working (spec story 5: open/record any
// folder even with empty history).
async function resolveTarget() {
    if (selected >= 0 && filtered[selected]) return filtered[selected].path;
    const typed = cleanPath(input.value.trim());
    if (typed && (await IsDirectory(typed))) return typed;
    return null;
}

// Enter behavior depends on the armed open method and the selected item:
//  - An action was armed with Left/Right arrows (activeAction >= 0): open
//    the resolved target with that software, like pressing its digit key.
//  - Default action, searching, and the selected item is already in the
//    history (Recorded): open it directly in Explorer — no need to re-record
//    something that is already at hand.
//  - Default action, searching, and the item is a fresh filesystem discovery
//    (or the box holds a path with no list entry): record it to the top of
//    Recent Folders WITHOUT opening (panel stays open, box clears).
//  - Default action, empty search box: open the selected folder in Explorer.
async function openDefault() {
    const path = await resolveTarget();
    if (!path) return;
    if (activeAction >= 0) {
        runAndRefresh(() => OpenWith(path, activeAction));
        return;
    }
    if (input.value.trim() !== '') {
        const item = selected >= 0 ? filtered[selected] : null;
        // Already-recorded items open directly; only fresh discoveries
        // (or a bare typed path) get recorded.
        if (!item || !item.recorded) {
            try {
                await Record(path);
                input.value = '';
                await load();
                input.focus();
            } catch (err) {
                console.error('record failed:', err);
                showEmpty('无法记录该文件夹', String(err));
            }
            return;
        }
        // Recorded item -> fall through to open it.
    }
    // Empty search box -> open the selected folder in Explorer.
    runAndRefresh(() => Open(path));
}

// A digit key: open the resolved target with software[index], if that many
// apps are configured. Keys beyond the list are ignored.
async function openWithKey(index) {
    if (index >= software.length) return;
    const path = await resolveTarget();
    if (!path || !availableActionIndexes(path).includes(index)) return;
    runAndRefresh(() => OpenWith(path, index));
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
    // Hide via HideAfterOpen rather than WindowHide: it tells the panel
    // not to restore the pre-panel foreground window, so the process we
    // just launched (Explorer / PowerShell / IDE) keeps the foreground.
    HideAfterOpen();
    dismiss();
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
    activeAction = -1;
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
            ? '回车打开已记录项 / 记录新发现；← → 选择打开方式（1-9 也可）'
            : '回车打开已记录项，新路径回车记录';
        showEmpty('还没有历史记录', hint);
        return;
    }
    if (!filtered.length) {
        showEmpty('没有匹配的文件夹', '输入完整路径按回车记录；← → 选择打开方式后回车打开');
        return;
    }
    empty.classList.add('hidden');
    const frag = document.createDocumentFragment();
    filtered.forEach((folder, i) => {
        const li = document.createElement('li');
        li.dataset.index = String(i);
        li.title = folder.path;
        if (!folder.recorded) li.classList.add('unrecorded');

        const info = document.createElement('div');
        info.className = 'info';

        const name = document.createElement('span');
        name.className = 'name';
        name.textContent = folder.name || folder.path;
        info.appendChild(name);

        if (!folder.recorded) {
            const tag = document.createElement('span');
            tag.className = 'tag-new';
            tag.textContent = '新';
            info.appendChild(tag);
        }

        const path = document.createElement('span');
        path.className = 'path';
        path.textContent = folder.path;
        info.appendChild(path);

        li.appendChild(info);

        if (software.length) {
            const actions = document.createElement('span');
            actions.className = 'actions';
            const allowed = availableActionIndexes(folder);
            software.forEach((sw, si) => {
                const badge = document.createElement('span');
                badge.className = 'sw';
                const disabled = !allowed.includes(si);
                if (disabled) badge.classList.add('disabled');
                if (!disabled && i === selected && si === activeAction) badge.classList.add('active');
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

        if (i === selected) {
            li.classList.add('selected');
            // The left accent bar marks the DEFAULT open method: it is
            // shown while no app is armed (activeAction = -1) and hidden
            // once an app badge is armed with Left/Right arrows.
            if (activeAction === -1) li.classList.add('armed-default');
        }
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

// --- Settings view ---

// showView toggles which view is rendered. It is called on panel-opened /
// settings-opened and on the «返回面板» button.
function showView(v) {
    view = v;
    panel.classList.toggle('hidden', v !== 'panel');
    settingsView.classList.toggle('hidden', v !== 'settings');
    hideSettingsMsg();
}

// loadSettings fills the form from the persisted Settings. It does NOT focus
// the field: the settings view is read-mostly and the user edits by clicking
// in, so opening it should not pre-select / highlight a value.
async function loadSettings() {
    try {
        const s = await GetSettings();
        historyLimit.value = String(s.historyLimit);
        autostart.checked = !!s.autoStart;
        resetSwForm();
        renderSoftwareList(s.software || []);
    } catch (err) {
        console.error('GetSettings failed:', err);
        showSettingsMsg('读取设置失败: ' + err, true);
    }
}

// applySettings reads the form, validates client-side, and calls the binding
// which persists + applies the cap and the registry entry live. Called on
// every change (no Save button — edits save immediately).
async function applySettings() {
    const limit = parseInt(historyLimit.value, 10);
    if (!Number.isFinite(limit) || limit < 1) {
        showSettingsMsg('历史上限必须是大于 0 的整数', true);
        return;
    }
    try {
        await SaveSettings(limit, autostart.checked);
        hideSettingsMsg();
    } catch (err) {
        console.error('SaveSettings failed:', err);
        showSettingsMsg('保存失败: ' + err, true);
    }
}

function showSettingsMsg(text, isError) {
    settingsMsg.textContent = text;
    settingsMsg.classList.toggle('error', !!isError);
    settingsMsg.classList.remove('hidden');
}

function hideSettingsMsg() {
    settingsMsg.classList.add('hidden');
    settingsMsg.textContent = '';
}

backBtn.addEventListener('click', () => {
    showView('panel');
    input.focus();
});

// Edits save immediately: a change on either control (number input loses
// focus / Enter, checkbox toggled) applies the settings live.
historyLimit.addEventListener('change', applySettings);
autostart.addEventListener('change', applySettings);

// --- Software list CRUD ---

// renderSoftwareList draws the persisted entries with inline edit/delete and
// shows the ordinal key (1-9) each maps to; beyond 9 there is no key, only
// mouse click (spec: keys 1-9).
function renderSoftwareList(list) {
    swList.innerHTML = '';
    if (!list.length) {
        const li = document.createElement('li');
        li.className = 'sw-empty';
        li.textContent = '还没有软件，在下方添加。';
        swList.appendChild(li);
        return;
    }
    const frag = document.createDocumentFragment();
    list.forEach((sw, i) => {
        const li = document.createElement('li');
        li.dataset.index = String(i);

        if (i <= 8) {
            const key = document.createElement('kbd');
            key.textContent = String(i + 1);
            li.appendChild(key);
        }

        const info = document.createElement('span');
        info.className = 'sw-info';
        const name = document.createElement('span');
        name.className = 'sw-name';
        name.textContent = sw.name;
        info.appendChild(name);
        const exe = document.createElement('span');
        exe.className = 'sw-exe';
        exe.textContent = sw.exe + (sw.args && sw.args.length ? ' ' + sw.args.join(' ') : '');
        info.appendChild(exe);
        li.appendChild(info);

        const edit = document.createElement('button');
        edit.type = 'button';
        edit.className = 'link';
        edit.textContent = '编辑';
        edit.addEventListener('click', () => beginEdit(i, sw));
        li.appendChild(edit);

        const del = document.createElement('button');
        del.type = 'button';
        del.className = 'link danger';
        del.textContent = '删除';
        del.addEventListener('click', () => removeSoftware(i));
        li.appendChild(del);

        frag.appendChild(li);
    });
    swList.appendChild(frag);
}

// beginEdit loads an entry into the form for editing and flips the add
// button to «保存修改».
// beginEdit loads an entry into the form for editing, shows the form and
// flips the button to «保存修改».
function beginEdit(index, sw) {
    editIndex = index;
    swName.value = sw.name || '';
    swExe.value = sw.exe || '';
    swArgs.value = (sw.args && sw.args.length) ? sw.args.join(' ') : '';
    swAddBtn.textContent = '保存修改';
    swForm.classList.remove('hidden');
    swName.focus();
}

// beginAdd opens the form blank for a new entry.
function beginAdd() {
    editIndex = -1;
    swName.value = '';
    swExe.value = '';
    swArgs.value = '';
    swAddBtn.textContent = '保存';
    swForm.classList.remove('hidden');
    swName.focus();
}

// resetSwForm hides the form and clears it back to add mode (no selection).
function resetSwForm() {
    editIndex = -1;
    swName.value = '';
    swExe.value = '';
    swArgs.value = '';
    swAddBtn.textContent = '保存';
    swForm.classList.add('hidden');
}

// commitSoftware adds a new entry or updates the edited one from the form,
// then reloads the list. Args are space-split into a slice; {folder} is
// kept literal so the backend placeholder substitution still works.
async function commitSoftware() {
    const name = swName.value.trim();
    const exe = swExe.value.trim();
    if (!name || !exe) {
        showSettingsMsg('名称和路径不能为空', true);
        return;
    }
    const args = swArgs.value.trim() ? swArgs.value.trim().split(/\s+/) : [];
    try {
        if (editIndex >= 0) {
            await UpdateSoftware(editIndex, name, exe, args);
        } else {
            await AddSoftware(name, exe, args);
        }
        resetSwForm();
        await reloadSoftwareList();
        hideSettingsMsg();
    } catch (err) {
        console.error('commitSoftware failed:', err);
        showSettingsMsg('保存失败: ' + err, true);
    }
}

async function removeSoftware(index) {
    try {
        await DeleteSoftware(index);
        resetSwForm();
        await reloadSoftwareList();
        hideSettingsMsg();
    } catch (err) {
        console.error('DeleteSoftware failed:', err);
        showSettingsMsg('删除失败: ' + err, true);
    }
}

// reloadSoftwareList re-reads settings and redraws only the list, without
// disturbing the cap/autostart form fields.
async function reloadSoftwareList() {
    try {
        const s = await GetSettings();
        renderSoftwareList(s.software || []);
    } catch (err) {
        console.error('reloadSoftwareList failed:', err);
    }
}

swAddBtn.addEventListener('click', commitSoftware);
swNewBtn.addEventListener('click', beginAdd);
swCancelBtn.addEventListener('click', resetSwForm);

// Initial paint in case the webview loads before the first panel-opened.
load();
