package rfa

import (
	"context"
	"encoding/json"
	"fmt"
)

// FileContent holds a filename and its full text content.
// Returned by GetAllFiles.
type FileContent struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// FileMetadata holds info about a file without its content.
// Returned by GetFileMetadata and GetAllFileMetadata.
type FileMetadata struct {
	Filename  string `json:"filename"`
	Length    int    `json:"length"`
	Timestamp int64  `json:"timestamp"`
}

// ServerInfo holds info about one server in the game world.
// Returned by GetAllServers.
type ServerInfo struct {
	Hostname          string `json:"hostname"`
	HasAdminRights    bool   `json:"hasAdminRights"`
	PurchasedByPlayer bool   `json:"purchasedByPlayer"`
}

// PushFile writes a file to a server inside the game.
// filename must start with / (e.g. "/larry-agent.js").
func (s *Server) PushFile(ctx context.Context, server, filename, content string) error {
	_, err := s.Call(ctx, "pushFile", map[string]string{
		"server":   server,
		"filename": filename,
		"content":  content,
	})
	return err
}

// DeleteFile removes a file from a server inside the game.
func (s *Server) DeleteFile(ctx context.Context, server, filename string) error {
	_, err := s.Call(ctx, "deleteFile", map[string]string{
		"server":   server,
		"filename": filename,
	})
	return err
}

// GetFile reads a file's content from a server inside the game.
func (s *Server) GetFile(ctx context.Context, server, filename string) (string, error) {
	raw, err := s.Call(ctx, "getFile", map[string]string{
		"server":   server,
		"filename": filename,
	})
	if err != nil {
		return "", err
	}
	var content string
	if err := json.Unmarshal(raw, &content); err != nil {
		return "", fmt.Errorf("getFile: bad response: %w", err)
	}
	return content, nil
}

// GetFileNames returns a list of all filenames on a server.
func (s *Server) GetFileNames(ctx context.Context, server string) ([]string, error) {
	raw, err := s.Call(ctx, "getFileNames", map[string]string{
		"server": server,
	})
	if err != nil {
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, fmt.Errorf("getFileNames: bad response: %w", err)
	}
	return names, nil
}

// GetAllFiles returns every file and its content from a server.
func (s *Server) GetAllFiles(ctx context.Context, server string) ([]FileContent, error) {
	raw, err := s.Call(ctx, "getAllFiles", map[string]string{
		"server": server,
	})
	if err != nil {
		return nil, err
	}
	var files []FileContent
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil, fmt.Errorf("getAllFiles: bad response: %w", err)
	}
	return files, nil
}

// GetFileMetadata returns metadata (size, timestamp) for a single file.
func (s *Server) GetFileMetadata(ctx context.Context, server, filename string) (FileMetadata, error) {
	raw, err := s.Call(ctx, "getFileMetadata", map[string]string{
		"server":   server,
		"filename": filename,
	})
	if err != nil {
		return FileMetadata{}, err
	}
	var meta FileMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return FileMetadata{}, fmt.Errorf("getFileMetadata: bad response: %w", err)
	}
	return meta, nil
}

// GetAllFileMetadata returns metadata for every file on a server.
func (s *Server) GetAllFileMetadata(ctx context.Context, server string) ([]FileMetadata, error) {
	raw, err := s.Call(ctx, "getAllFileMetadata", map[string]string{
		"server": server,
	})
	if err != nil {
		return nil, err
	}
	var meta []FileMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, fmt.Errorf("getAllFileMetadata: bad response: %w", err)
	}
	return meta, nil
}

// GetAllServers returns every server in the game world.
func (s *Server) GetAllServers(ctx context.Context) ([]ServerInfo, error) {
	raw, err := s.Call(ctx, "getAllServers", map[string]string{})
	if err != nil {
		return nil, err
	}
	var servers []ServerInfo
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil, fmt.Errorf("getAllServers: bad response: %w", err)
	}
	return servers, nil
}

// CalculateRam returns the RAM cost in GB of a script on a server.
func (s *Server) CalculateRam(ctx context.Context, server, filename string) (float64, error) {
	raw, err := s.Call(ctx, "calculateRam", map[string]string{
		"server":   server,
		"filename": filename,
	})
	if err != nil {
		return 0, err
	}
	var gb float64
	if err := json.Unmarshal(raw, &gb); err != nil {
		return 0, fmt.Errorf("calculateRam: bad response: %w", err)
	}
	return gb, nil
}

// GetDefinitionFile fetches Bitburner's NetscriptDefinitions.d.ts.
func (s *Server) GetDefinitionFile(ctx context.Context) (string, error) {
	raw, err := s.Call(ctx, "getDefinitionFile", map[string]string{})
	if err != nil {
		return "", err
	}
	var content string
	if err := json.Unmarshal(raw, &content); err != nil {
		return "", fmt.Errorf("getDefinitionFile: bad response: %w", err)
	}
	return content, nil
}

// GetSaveFile returns the full game save as a base64-encoded string.
func (s *Server) GetSaveFile(ctx context.Context) (string, error) {
	raw, err := s.Call(ctx, "getSaveFile", map[string]string{})
	if err != nil {
		return "", err
	}
	var save string
	if err := json.Unmarshal(raw, &save); err != nil {
		return "", fmt.Errorf("getSaveFile: bad response: %w", err)
	}
	return save, nil
}
