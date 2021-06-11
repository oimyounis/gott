package gott

import (
	"log"
	"net/http"

	"github.com/oimyounis/gott/utils"

	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

func onRequestHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrader", err)
		return
	}

	if !GOTT.invokeOnSocketOpen(conn.UnderlyingConn()) {
		_ = conn.Close()
		return
	}

	log.Printf("Accepted connection from %v", conn.RemoteAddr().String())

	c := &Client{
		wsConnection: conn,
		connected:    atomicBool{val: true},
	}
	go c.listen()
}

type webSocketsServer struct {
	config Config
}

func newWebSocketsServer(c Config) *webSocketsServer {
	upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		Subprotocols:    []string{"mqtt"},
		CheckOrigin: func(r *http.Request) bool {
			if len(c.WebSockets.Origins) == 0 {
				return true
			}

			origin := r.Header.Get("origin")
			if origin == "" && c.WebSockets.RejectEmptyOrigin {
				return false
			}
			return utils.StringInSlice(origin, c.WebSockets.Origins)
		},
	}
	http.HandleFunc(c.WebSockets.Path, onRequestHandler)
	return &webSocketsServer{c}
}

func (wss *webSocketsServer) Listen() {
	err := http.ListenAndServe(wss.config.WebSockets.Listen, nil)
	if err != nil {
		log.Fatal("ListenAndServe WS: ", err)
	}
}
func (wss *webSocketsServer) ListenTLS() {
	err := http.ListenAndServeTLS(wss.config.WebSockets.WSS.Listen, wss.config.WebSockets.WSS.Cert, wss.config.WebSockets.WSS.Key, nil)
	if err != nil {
		log.Fatal("ListenAndServeTLS WS: ", err)
	}
}
