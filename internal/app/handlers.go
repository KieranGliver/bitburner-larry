package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/KieranGliver/bitburner-larry/internal/communication"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
	"github.com/sgtdi/fswatcher"
)

func (a *App) Start() {
	entries, err := os.ReadDir("scripts/src")
	if err != nil {
		a.P.Send(logger.Error("read scripts/src: " + err.Error()))
		return
	}
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if entry.IsDir() || (ext != ".txt" && ext != ".script") {
			continue
		}
		content, err := os.ReadFile(filepath.Join("scripts/src", entry.Name()))
		if err != nil {
			a.P.Send(logger.Warn("read " + entry.Name() + ": " + err.Error()))
			continue
		}
		os.WriteFile(filepath.Join("scripts/dist", entry.Name()), content, 0644)
	}
}

func (a *App) OnConnect(b *communication.BitburnerConn) {
	a.syncing.Store(true)
	defer a.syncing.Store(false)

	a.State.SetConn(b)
	conn := a.State.Conn()

	ctx := context.Background()

	// 0. Clear stale inbox/outbox files from any previous session
	if files, err := conn.GetFileNames(ctx, "home"); err == nil {
		for _, f := range files {
			norm := strings.TrimPrefix(f, "/")
			if strings.HasPrefix(norm, "inbox/") || strings.HasPrefix(norm, "outbox/") {
				if err := conn.DeleteFile(ctx, "home", f); err != nil {
					a.P.Send(logger.Warn("clear stale " + f + ": " + err.Error()))
				}
			}
		}
	}

	// 1. Pull the Netscript type definitions from the game
	dts, err := conn.GetDefinitionFile(ctx)
	if err != nil {
		a.P.Send(logger.Error("GetDefinitionFile: " + err.Error()))
		return
	}
	if err := os.WriteFile("scripts/NetscriptDefinitions.d.ts", []byte(dts), 0644); err != nil {
		a.P.Send(logger.Error("write NetscriptDefinitions.d.ts: " + err.Error()))
		return
	}
	a.P.Send(logger.Info("synced NetscriptDefinitions.d.ts"))

	// 2. Seed local dist with game files that don't exist locally yet
	gameFiles, err := conn.GetAllFiles(ctx, "home")
	if err != nil {
		a.P.Send(logger.Error("GetAllFiles: " + err.Error()))
		return
	}
	seeded := 0
	seededFiles := []string{}
	for _, f := range gameFiles {
		if strings.HasPrefix(f.Filename, "inbox/") || strings.HasPrefix(f.Filename, "outbox/") {
			continue
		}
		localPath := filepath.Join("scripts/dist", f.Filename)
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			os.WriteFile(localPath, []byte(f.Content), 0644)
			seededFiles = append(seededFiles, f.Filename)
			seeded++
		}
	}

	// 3. Push all local dist files to the game (local is truth)
	distEntries, err := os.ReadDir("scripts/dist")
	if err != nil {
		a.P.Send(logger.Error("read scripts/dist: " + err.Error()))
		return
	}
	pushed := 0
	pushedFiles := []string{}
	for _, entry := range distEntries {
		if entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), "inbox/") || strings.HasPrefix(entry.Name(), "outbox/") {
			continue
		}
		localPath := filepath.Join("scripts/dist", entry.Name())
		content, err := os.ReadFile(localPath)
		if err != nil {
			a.P.Send(logger.Warn("read " + entry.Name() + ": " + err.Error()))
			continue
		}
		if err := conn.PushFile(ctx, "home", entry.Name(), string(content)); err != nil {
			a.P.Send(logger.Warn("push " + entry.Name() + ": " + err.Error()))
			continue
		}
		pushedFiles = append(pushedFiles, entry.Name())
		pushed++
	}

	var detail strings.Builder
	if len(seededFiles) > 0 {
		detail.WriteString("seeded:\n")
		for _, f := range seededFiles {
			detail.WriteString("  " + f + "\n")
		}
	}
	if len(pushedFiles) > 0 {
		detail.WriteString("pushed:\n")
		for _, f := range pushedFiles {
			detail.WriteString("  " + f + "\n")
		}
	}
	a.P.Send(logger.InfoDetail(
		fmt.Sprintf("sync complete: seeded %d, pushed %d", seeded, pushed),
		strings.TrimRight(detail.String(), "\n"),
	))

}

func hasContentChange(event fswatcher.WatchEvent) bool {
	return slices.Contains(event.Flags, "Modified") || slices.Contains(event.Flags, "Created")
}

func (a *App) OnEventDist(event fswatcher.WatchEvent) {
	conn := a.State.conn
	if conn == nil || a.syncing.Load() {
		return
	}

	filename := filepath.Base(event.Path)
	ext := filepath.Ext(event.Path)
	if ext != ".js" && ext != ".txt" && ext != ".script" {
		return
	}
	if strings.Contains(event.Path, "/inbox/") || strings.Contains(event.Path, "/outbox/") {
		return
	}

	if slices.Contains(event.Types, fswatcher.EventRemove) || slices.Contains(event.Types, fswatcher.EventRename) {
		conn.DeleteFile(context.Background(), "home", filename)
		a.P.Send(logger.Info("[filesync] deleted " + filename + " from Bitburner"))
		return
	}

	info, err := os.Stat(event.Path)
	if err != nil || info.IsDir() {
		return
	}

	if !hasContentChange(event) {
		return
	}

	content, err := os.ReadFile(event.Path)
	if err != nil {
		a.P.Send(logger.Warn("[filesync] read " + event.Path + ": " + err.Error()))
		return
	}

	if err := conn.PushFile(context.Background(), "home", filename, string(content)); err != nil {
		a.P.Send(logger.Error("[filesync] push " + filename + ": " + err.Error()))
		return
	}

	a.P.Send(logger.Info("[filesync] pushed " + filename + " to Bitburner"))
}

func (a *App) OnEventSrc(event fswatcher.WatchEvent) {

	ext := filepath.Ext(event.Path)
	if ext != ".ts" && ext != ".txt" && ext != ".script" {
		return
	}

	base := strings.TrimSuffix(filepath.Base(event.Path), ext)

	if slices.Contains(event.Types, fswatcher.EventRemove) || slices.Contains(event.Types, fswatcher.EventRename) {
		os.Remove(filepath.Join("scripts/dist", base+".js"))
		os.Remove(filepath.Join("scripts/dist", base+".txt"))
		os.Remove(filepath.Join("scripts/dist", base+".script"))
		// OnEventDist will fire for the removed files and delete from Bitburner
		return
	}

	if ext != ".txt" && ext != ".script" {
		return
	}

	info, err := os.Stat(event.Path)
	if err != nil || info.IsDir() {
		return
	}

	if !hasContentChange(event) {
		return
	}

	content, err := os.ReadFile(event.Path)
	if err != nil {
		a.P.Send(logger.Warn("[filesync] read " + event.Path + ": " + err.Error()))
		return
	}

	if err := os.WriteFile(filepath.Join("scripts/dist", base+ext), content, 0644); err != nil {
		a.P.Send(logger.Warn("[filesync] copy " + base + ext + " to dist: " + err.Error()))
	}
	// OnEventDist will pick up the change and push to Bitburner
}
