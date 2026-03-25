package communication

import (
	"context"
	"fmt"
	"io"
	"net/http"

	tea "charm.land/bubbletea/v2"
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
	Status websocketStatus
	conn   *websocket.Conn
}

func (b *BitburnerConn) Send(ctx context.Context, msg string) error {
	return b.conn.Write(ctx, websocket.MessageText, []byte(msg))
}

func Serve(port string, p *tea.Program) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleWS(w, r, p)
	}) // Bitburner daemon connects here
	http.HandleFunc("/done", handleDone) // scripts POST here when finished

	http.ListenAndServe(":"+port, nil)
}

var active *BitburnerConn // nil means nobody connected

func handleWS(w http.ResponseWriter, r *http.Request, p *tea.Program) {
	// Upgrade to websocket
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		fmt.Println("WS: upgrade failed:", err)
	}
	defer conn.CloseNow()

	active = &BitburnerConn{Status: Connected, conn: conn}
	defer func() { active = nil }() // clear it when they disconnect
	p.Send(BitburnerConnected{Conn: active})

	for {
		_, msg, err := conn.Read(r.Context())
		if err != nil {
			active.Status = Disconnected
			p.Send(BitburnerDisconnected{})
			return
		}
		fmt.Println("received:", string(msg))
	}
}

func handleDone(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	fmt.Println("script finished:", string(body))
	w.WriteHeader(http.StatusOK)
}
