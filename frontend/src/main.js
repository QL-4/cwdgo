import './style.css';

import { GetRecentFolders } from '../wailsjs/go/main/App';
import { EventsOn, WindowHide } from '../wailsjs/runtime/runtime';

const input = document.getElementById('search');
const empty = document.getElementById('empty');
const results = document.getElementById('results');

// The panel opens (hotkey or tray): focus the search box immediately.
// (Click-outside-to-close is handled in Go via WM_ACTIVATE, not here: the
// webview's blur event fires spuriously because WebView2 bounces focus
// between the host window and its Chromium helper windows.)
EventsOn('panel-opened', () => {
    input.focus();
    input.select();
});

// Esc closes the panel.
window.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        WindowHide();
    }
});

// Render the Recent Folders list; the empty state explains the first run.
// (Search filtering and keyboard navigation arrive in a later ticket.)
GetRecentFolders().then((entries) => {
    if (!entries.length) {
        empty.classList.remove('hidden');
        return;
    }
    for (const entry of entries) {
        const li = document.createElement('li');
        li.textContent = entry.Name;
        li.title = entry.Path;
        results.appendChild(li);
    }
}).catch((err) => {
    console.error('GetRecentFolders failed:', err);
});
