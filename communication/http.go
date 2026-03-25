package communication

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/KieranGliver/bitburner-larry/logger"
	"github.com/coder/websocket"
)

type BitburnerConnected struct {
	Conn *BitburnerConn
}
type BitburnerDisconnected struct{}

type websocketStatus uint

const (
	Connected websocketStatus = iota
	Disconnected
)

type BitburnerConn struct {
	Status  websocketStatus
	conn    *websocket.Conn
	nextID  int64
	pending map[int64]chan RpcResponse
	mu      sync.Mutex
}

func (b *BitburnerConn) Send(ctx context.Context, msg string) error {
	return b.conn.Write(ctx, websocket.MessageText, []byte(msg))
}

func Serve(port string, p *tea.Program, onConnect func(*BitburnerConn, *tea.Program)) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleWS(w, r, p, onConnect)
	})
	http.HandleFunc("/done", func(w http.ResponseWriter, r *http.Request) {
		handleDone(w, r, p)
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("HTTP server error: %v\n", err)
	}
}

func (b *BitburnerConn) Call(ctx context.Context, method string) (json.RawMessage, error) {
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	ch := make(chan RpcResponse, 1)
	b.pending[id] = ch
	b.mu.Unlock()

	req := RpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: map[string]any{}}
	data, _ := json.Marshal(req)
	if err := b.conn.Write(ctx, websocket.MessageText, data); err != nil {
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp.Result, nil
	case <-ctx.Done():
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, ctx.Err()
	}
}

var active *BitburnerConn

func handleWS(w http.ResponseWriter, r *http.Request, p *tea.Program, onConnect func(*BitburnerConn, *tea.Program)) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		p.Send(logger.Error("WS upgrade failed: " + err.Error()))
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(10 * 1024 * 1024) // 10MB

	active = &BitburnerConn{
		Status:  Connected,
		conn:    conn,
		pending: make(map[int64]chan RpcResponse),
	}
	go onConnect(active, p)
	defer func() { active = nil }()
	p.Send(BitburnerConnected{Conn: active})
	p.Send(logger.Info("Bitburner connected"))

	for {
		_, msg, err := conn.Read(r.Context())
		if err != nil {
			active.Status = Disconnected
			p.Send(BitburnerDisconnected{})
			p.Send(logger.Info("Bitburner disconnected: " + err.Error()))
			return
		}

		var resp RpcResponse
		json.Unmarshal(msg, &resp)

		if resp.ID != 0 {
			// response to one of our calls — route it back to Call()
			active.mu.Lock()
			ch, ok := active.pending[resp.ID]
			if ok {
				delete(active.pending, resp.ID)
			}
			active.mu.Unlock()
			if ok {
				ch <- resp
			}
		} else {
			// unsolicited message from Bitburner — log it for now
			p.Send(logger.Info("received: " + string(msg)))
		}
	}
}

func handleDone(w http.ResponseWriter, r *http.Request, p *tea.Program) {
	body, _ := io.ReadAll(r.Body)
	p.Send(logger.Info("script finished: " + string(body)))
	w.WriteHeader(http.StatusOK)
}
