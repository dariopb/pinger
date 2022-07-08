package pinger

import (
	"net/http"

	"github.com/dariopb/pinger/console"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

func handleconsolews(cmd string, w http.ResponseWriter, r *http.Request) {
	l := log.WithField("remoteaddr", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.WithError(err).Error("Unable to upgrade connection")
		return
	}

	l.WithField("cmd", cmd).Info("handle console cmd")
	d := webSocketConsoleConn{Conn: conn}

	wc, err := console.NewWinConsole(cmd, &d)
	if err != nil {
		log.Errorf("NewWinConsole: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}

	d.resize = func(cols uint16, rows uint16) {
		err := wc.Resize(cols, rows)
		if err != nil {
		}
	}
	d.close = func() {
		wc.Close()
	}

	wc.Wait()

	log.Infof("Windows shell handler terminated.")
	conn.WriteMessage(websocket.TextMessage, []byte("Windows shell handler terminated."))

	conn.Close()
}
