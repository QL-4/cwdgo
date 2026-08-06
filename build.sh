#!/usr/bin/env bash
# build.sh — 构建并重启 cwdgo，避免旧进程占用热键导致弹窗。
#
# 流程：杀旧进程 → wails build → 启动新 exe（后台）
# 用法: ./build.sh
set -euo pipefail
export PATH="/c/Program Files/Go/bin:/c/Users/jerem/go/bin:$PATH"

cd "$(dirname "$0")"

echo "=== 杀掉旧 cwdgo 进程 ==="
powershell.exe -NoProfile -Command "Get-Process cwdgo -ErrorAction SilentlyContinue | Stop-Process -Force" 2>/dev/null || true
sleep 1

echo "=== wails build ==="
wails.exe build 2>&1 | tail -3

echo "=== 启动新 cwdgo ==="
./build/bin/cwdgo.exe >/dev/null 2>&1 &
disown 2>/dev/null || true
sleep 2

echo "=== 当前进程 ==="
powershell.exe -NoProfile -Command "Get-Process cwdgo -ErrorAction SilentlyContinue | Select-Object Id,StartTime | Format-Table -AutoSize" 2>/dev/null
echo "done"
