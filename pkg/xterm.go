package pinger

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type windowSize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
	X    uint16
	Y    uint16
}
type webSocketConsoleConn struct {
	Conn *websocket.Conn

	hpc    uintptr //syscall.Handle
	resize func(x uint16, y uint16)
	close  func()
}

func (d *webSocketConsoleConn) Write(p []byte) (n int, err error) {
	err = d.Conn.WriteMessage(websocket.BinaryMessage, p)
	return len(p), err
}

func (d *webSocketConsoleConn) Read(p []byte) (n int, err error) {
	for {
		messageType, reader, err := d.Conn.NextReader()
		if err != nil {
			log.Errorf("Read error from websocket: %s", err.Error())
			d.close()
			return 0, err
		}

		if messageType == websocket.TextMessage {
			d.Conn.WriteMessage(websocket.TextMessage, []byte("Unexpected text message"))
			return 0, nil
		}

		dataTypeBuf := make([]byte, 1)
		read, err := reader.Read(dataTypeBuf)
		if err != nil {
			log.Errorf("Read error from websocket: %s", err.Error())
			d.close()
			d.Conn.WriteMessage(websocket.TextMessage, []byte("Unable to read message type from reader"))
			return 0, err
		}

		if read != 1 {
			return 0, nil
		}

		switch dataTypeBuf[0] {
		case 0:
			n, err := reader.Read(p)
			return n, err
		case 1:
			decoder := json.NewDecoder(reader)
			resizeMessage := windowSize{}
			err := decoder.Decode(&resizeMessage)
			if err != nil {
				//conn.WriteMessage(websocket.TextMessage, []byte("Error decoding resize message: "+err.Error()))
				continue
			}
			log.WithField("resizeMessage", resizeMessage).Info("Resizing terminal")

			if d.resize != nil {
				d.resize(resizeMessage.Cols, resizeMessage.Rows)
			}
			continue
		default:
			d.close()
		}

		continue
	}
}

func (d *webSocketConsoleConn) Close() error {
	return nil
}
