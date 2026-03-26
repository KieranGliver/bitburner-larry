package communication

import (
	"context"
	"encoding/json"
	"fmt"
)

type RpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type RpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   string          `json:"error,omitempty"`
}

type Server struct {
	Hostname          string `json:"hostname"`
	HasAdminRights    bool   `json:"hasAdminRights"`
	PurchasedByPlayer bool   `json:"purchasedByPlayer"`
}

type FileContent struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

type FileMetadata struct {
	Filename  string `json:"filename"`
	Length    int    `json:"length"`
	Timestamp int64  `json:"timestamp"`
}

func (b *BitburnerConn) PushFile(ctx context.Context, server, filename, content string) error {
	_, err := b.Call(ctx, "pushFile", map[string]string{
		"server":   server,
		"filename": filename,
		"content":  content,
	})
	return err
}

func (b *BitburnerConn) DeleteFile(ctx context.Context, server, filename string) error {
	_, err := b.Call(ctx, "deleteFile", map[string]string{
		"server":   server,
		"filename": filename,
	})
	return err
}

func (b *BitburnerConn) GetFile(ctx context.Context, server, filename string) (string, error) {
	raw, err := b.Call(ctx, "getFile", map[string]string{
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

func (b *BitburnerConn) GetFileNames(ctx context.Context, server string) ([]string, error) {
	raw, err := b.Call(ctx, "getFileNames", map[string]string{
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

func (b *BitburnerConn) GetAllFiles(ctx context.Context, server string) ([]FileContent, error) {
	raw, err := b.Call(ctx, "getAllFiles", map[string]string{
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

func (b *BitburnerConn) GetFileMetadata(ctx context.Context, server, filename string) (FileMetadata, error) {
	raw, err := b.Call(ctx, "getFileMetadata", map[string]string{
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

func (b *BitburnerConn) GetAllFileMetadata(ctx context.Context, server string) ([]FileMetadata, error) {
	raw, err := b.Call(ctx, "getAllFileMetadata", map[string]string{
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

func (b *BitburnerConn) GetAllServers(ctx context.Context) ([]Server, error) {
	raw, err := b.Call(ctx, "getAllServers", map[string]string{})
	if err != nil {
		return nil, err
	}
	var servers []Server
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil, fmt.Errorf("getAllServers: bad response: %w", err)
	}
	return servers, nil
}

func (b *BitburnerConn) CalculateRam(ctx context.Context, server, filename string) (float64, error) {
	raw, err := b.Call(ctx, "calculateRam", map[string]string{
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

func (b *BitburnerConn) GetDefinitionFile(ctx context.Context) (string, error) {
	raw, err := b.Call(ctx, "getDefinitionFile", map[string]string{})
	if err != nil {
		return "", err
	}
	var content string
	if err := json.Unmarshal(raw, &content); err != nil {
		return "", fmt.Errorf("getDefinitionFile: bad response: %w", err)
	}
	return content, nil
}

func (b *BitburnerConn) GetSaveFile(ctx context.Context) (string, error) {
	raw, err := b.Call(ctx, "getSaveFile", map[string]string{})
	if err != nil {
		return "", err
	}
	var save string
	if err := json.Unmarshal(raw, &save); err != nil {
		return "", fmt.Errorf("getSaveFile: bad response: %w", err)
	}
	return save, nil
}
