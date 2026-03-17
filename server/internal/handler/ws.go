package handler

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/shinyoung/investment/internal/ws"
)

func HandleWebSocket(hub *ws.Hub) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("failed to upgrade websocket", "error", err)
			return
		}

		client := ws.NewClient(hub, conn)
		hub.Register() <- client

		go client.WritePump()
		client.ReadPump()
	}
}
