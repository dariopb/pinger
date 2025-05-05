package pinger

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

func handleconsolews(xterObj *XtermObj, w http.ResponseWriter, r *http.Request) {
	l := log.WithField("remoteaddr", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.WithError(err).Error("Unable to upgrade connection")
		return
	}

	d := webSocketConsoleConn{Conn: conn}
	var stdIn io.Reader
	var stdOut io.Writer

	if xterObj.SshTarget != "" {
		l.WithField("sshTarget", xterObj.SshTarget).Infof("SSH target detected: %s", xterObj.SshTarget)

		sshClient, err := NewSshClient(xterObj.SshTarget)
		if err != nil {
			l.WithError(err).Error("Unable to create SSH client")
			conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
			return
		}

		stdIn = sshClient.sessOut
		stdOut = sshClient.sessIn
		d.resize = sshClient.Resize

		d.close = func() {
			sshClient.Close()
		}

		waitgroup := &sync.WaitGroup{}
		waitgroup.Add(1)

		go func() {
			io.Copy(&d, stdIn)
		}()
		go func() {
			io.Copy(stdOut, &d)
		}()

		go func() {
			sshClient.sess.Wait()
			waitgroup.Done()
		}()

		waitgroup.Wait()

		sshClient.Close()
	} else {
		l.WithField("cmd", xterObj.Cmd).Info("handle console cmd")

		cmd := exec.Command(xterObj.Cmd) // -l
		cmd.Env = append(os.Environ(), "TERM=xterm", "LANG=C")

		tty, err := pty.Start(cmd)
		if err != nil {
			l.WithError(err).Error("Unable to start pty/cmd")
			conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
			return
		}
		stdIn = tty
		stdOut = tty

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

		go io.Copy(&d, stdIn)
		go io.Copy(stdOut, &d)

		cmd.Process.Wait()
	}

	log.Infof("Linux shell handler terminated.")
	conn.WriteMessage(websocket.TextMessage, []byte("Linux shell handler terminated."))

	conn.Close()
}

func wwwhandleconsolews(xterObj *XtermObj, w http.ResponseWriter, r *http.Request) {
	l := log.WithField("remoteaddr", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		l.WithError(err).Error("Unable to upgrade connection")
		return
	}

	cmd := exec.Command(xterObj.Cmd) // -l
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
