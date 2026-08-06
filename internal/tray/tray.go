// Package tray runs the systray icon and its menu («打开面板» / «退出»).
// It runs on its own OS thread so the Wails main thread stays free.
package tray

import (
	"github.com/getlantern/systray"

	"cwdgo/internal/applog"
	"cwdgo/internal/icon"
)

// Run starts the tray icon. onOpen is invoked when the user picks
// «打开面板», onExit when the user picks «退出» (after the tray has fully
// shut down). All run on the tray's own thread.
func Run(onOpen, onSettings, onExit func()) {
	go func() {
		systray.Run(func() {
			applog.Log("tray: ready")
			systray.SetIcon(icon.TrayICO())
			systray.SetTooltip("cwdgo — Recent Folders Launcher")

			mOpen := systray.AddMenuItem("打开面板", "打开启动面板")
			mSettings := systray.AddMenuItem("设置", "打开设置")
			mQuit := systray.AddMenuItem("退出", "退出 cwdgo")
			systray.AddSeparator()

			for {
				select {
				case <-mOpen.ClickedCh:
					applog.Log("tray: 打开面板 clicked")
					onOpen()
				case <-mSettings.ClickedCh:
					applog.Log("tray: 设置 clicked")
					onSettings()
				case <-mQuit.ClickedCh:
					applog.Log("tray: 退出 clicked")
					systray.Quit()
					onExit()
					return
				}
			}
		}, func() {})
	}()
}
