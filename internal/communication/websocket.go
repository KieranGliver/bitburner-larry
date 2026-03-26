package communication

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/KieranGliver/bitburner-larry/internal/logger"
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
	Status      websocketStatus
	conn        *websocket.Conn
	nextID      int64
	pending     map[int64]chan RpcResponse
	httpPending map[string]chan string
	mu          sync.Mutex
}

// RegisterHTTP creates a channel that will receive the body of the first /done
// POST whose JSON contains `"id": id`. Call this before sending the request.
func (b *BitburnerConn) RegisterHTTP(id string) <-chan string {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan string, 1)
	b.httpPending[id] = ch
	return ch
}

func (b *BitburnerConn) resolveHTTP(id, data string) bool {
	b.mu.Lock()
	ch, ok := b.httpPending[id]
	if ok {
		delete(b.httpPending, id)
	}
	b.mu.Unlock()
	if ok {
		ch <- data
	}
	return ok
}

func Serve(port string, p *tea.Program, onConnect func(*BitburnerConn)) {
	var (
		activeConn *BitburnerConn
		activeMu   sync.Mutex
	)
	setConn := func(c *BitburnerConn) {
		activeMu.Lock()
		activeConn = c
		activeMu.Unlock()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleWS(w, r, p, func(c *BitburnerConn) {
			setConn(c)
			onConnect(c)
		})
		setConn(nil) // handleWS blocks until disconnect
	})
	mux.HandleFunc("/done", func(w http.ResponseWriter, r *http.Request) {
		activeMu.Lock()
		conn := activeConn
		activeMu.Unlock()
		handleDone(w, r, p, conn)
	})

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		fmt.Printf("HTTP server error: %v\n", err)
	}
}

func (b *BitburnerConn) Call(ctx context.Context, method string, params ...any) (json.RawMessage, error) {
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	ch := make(chan RpcResponse, 1)
	b.pending[id] = ch
	b.mu.Unlock()

	var p any = map[string]any{}
	if len(params) > 0 {
		p = params[0]
	}
	req := RpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: p}
	data, _ := json.Marshal(req)
	if err := b.conn.Write(ctx, websocket.MessageText, data); err != nil {
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.Error != "" {
			return nil, fmt.Errorf("rpc error: %s", resp.Error)
		}
		return resp.Result, nil
	case <-ctx.Done():
		b.mu.Lock()
		delete(b.pending, id)
		b.mu.Unlock()
		return nil, ctx.Err()
	}
}

func handleWS(w http.ResponseWriter, r *http.Request, p *tea.Program, onConnect func(*BitburnerConn)) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		p.Send(logger.Error("WS upgrade failed: " + err.Error()))
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(10 * 1024 * 1024) // 10MB

	active := &BitburnerConn{
		Status:      Connected,
		conn:        conn,
		pending:     make(map[int64]chan RpcResponse),
		httpPending: make(map[string]chan string),
	}
	go onConnect(active)
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
		if err := json.Unmarshal(msg, &resp); err != nil {
			p.Send(logger.Warn("failed to parse message: " + err.Error() + " raw: " + string(msg)))
			continue
		}

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

func handleDone(w http.ResponseWriter, r *http.Request, p *tea.Program, conn *BitburnerConn) {
	body, _ := io.ReadAll(r.Body)
	if conn != nil {
		var envelope struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(body, &envelope) == nil && envelope.ID != "" {
			if conn.resolveHTTP(envelope.ID, string(body)) {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
	}
	p.Send(logger.Warn("script finished: " + string(body)))
	w.WriteHeader(http.StatusOK)
}
