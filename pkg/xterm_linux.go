package pinger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

func handleconsolews(cmdToRun string, w http.ResponseWriter, r *http.Request) {
	l := log.WithField("remoteaddr", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.WithError(err).Error("Unable to upgrade connection")
		return
	}

	l.WithField("cmd", cmdToRun).Info("handle console cmd")
	d := webSocketConsoleConn{Conn: conn}

	cmd := exec.Command(cmdToRun) // -l
	cmd.Env = append(os.Environ(), "TERM=xterm")
	cmd.Env = append(os.Environ(), "LANG=C")

	tty, err := pty.Start(cmd)
	if err != nil {
		l.WithError(err).Error("Unable to start pty/cmd")
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}

	defer func() {
		cmd.Process.Kill()
		cmd.Process.Wait()
		tty.Close()
		conn.Close()
	}()

	d.resize = func(cols uint16, rows uint16) {
		resizeMessage := windowSize{Rows: rows, Cols: cols}
		_, _, errno := syscall.Syscall(
			syscall.SYS_IOCTL,
			tty.Fd(),
			syscall.TIOCSWINSZ,
			uintptr(unsafe.Pointer(&resizeMessage)),
		)
		if errno != 0 {
			l.WithError(syscall.Errno(errno)).Error("Unable to resize terminal")
		}
	}
	d.close = func() {
		cmd.Process.Kill()
		tty.Close()
	}

	go io.Copy(&d, tty)
	go io.Copy(tty, &d)

	cmd.Process.Wait()

	log.Infof("Linux shell handler terminated.")
	conn.WriteMessage(websocket.TextMessage, []byte("Linux shell handler terminated."))

	conn.Close()
}

func wwwhandleconsolews(cmdToRun string, w http.ResponseWriter, r *http.Request) {
	l := log.WithField("remoteaddr", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.WithError(err).Error("Unable to upgrade connection")
		return
	}

	cmd := exec.Command(cmdToRun) // -l
	cmd.Env = append(os.Environ(), "TERM=xterm")
	cmd.Env = append(os.Environ(), "LANG=C")

	tty, err := pty.Start(cmd)
	if err != nil {
		l.WithError(err).Error("Unable to start pty/cmd")
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Process.Wait()
		tty.Close()
		conn.Close()
	}()

	go func() {
		for {
			buf := make([]byte, 1024)
			read, err := tty.Read(buf)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
				l.WithError(err).Error("Unable to read from pty/cmd")
				return
			}
			fmt.Println(string(buf[:read]))
			conn.WriteMessage(websocket.BinaryMessage, buf[:read])
		}
	}()

	for {
		messageType, reader, err := conn.NextReader()
		if err != nil {
			l.WithError(err).Error("Unable to grab next reader")
			return
		}

		if messageType == websocket.TextMessage {
			l.Warn("Unexpected text message")
			conn.WriteMessage(websocket.TextMessage, []byte("Unexpected text message"))
			continue
		}

		dataTypeBuf := make([]byte, 1)
		read, err := reader.Read(dataTypeBuf)
		if err != nil {
			l.WithError(err).Error("Unable to read message type from reader")
			conn.WriteMessage(websocket.TextMessage, []byte("Unable to read message type from reader"))
			return
		}

		if read != 1 {
			l.WithField("bytes", read).Error("Unexpected number of bytes read")
			return
		}

		switch dataTypeBuf[0] {
		case 0:
			copied, err := io.Copy(tty, reader)
			if err != nil {
				l.WithError(err).Errorf("Error after copying %d bytes", copied)
			}
		case 1:
			decoder := json.NewDecoder(reader)
			resizeMessage := windowSize{}
			err := decoder.Decode(&resizeMessage)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte("Error decoding resize message: "+err.Error()))
				continue
			}
			log.WithField("resizeMessage", resizeMessage).Info("Resizing terminal")
			_, _, errno := syscall.Syscall(
				syscall.SYS_IOCTL,
				tty.Fd(),
				syscall.TIOCSWINSZ,
				uintptr(unsafe.Pointer(&resizeMessage)),
			)
			if errno != 0 {
				l.WithError(syscall.Errno(errno)).Error("Unable to resize terminal")
			}
		default:
			l.WithField("dataType", dataTypeBuf[0]).Error("Unknown data type")
		}
	}
}
