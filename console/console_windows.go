package console

import (
	"io"
	"syscall"
	"unsafe"

	"github.com/dariopb/pinger/syscalls"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

type WinConsole struct {
	conn io.ReadWriteCloser

	hpc      syscall.Handle
	procInfo windows.ProcessInformation
	fchan    chan bool
}

func NewWinConsole(cmd string, conn io.ReadWriteCloser) (*WinConsole, error) {
	wc := WinConsole{
		conn:  conn,
		fchan: make(chan bool),
	}
	var err error

	c := make(chan error)

	go func() {
		var cmdIn, cmdOut syscall.Handle
		var ptyIn, ptyOut syscall.Handle

		pSec := syscall.SecurityAttributes{}
		pSec.Length = uint32(unsafe.Sizeof(pSec))
		pSec.InheritHandle = 1
		if err = syscall.CreatePipe(&ptyIn, &cmdIn, &pSec, 1048576); err != nil {
			log.Errorf("CreatePipe: %v", err)
			c <- err
			return
		}
		defer syscall.CloseHandle(cmdIn)
		defer syscall.CloseHandle(ptyIn)
		if err = syscall.CreatePipe(&cmdOut, &ptyOut, &pSec, 1048576); err != nil {
			log.Errorf("CreatePipe: %v", err)
			c <- err
			return
		}
		defer syscall.CloseHandle(cmdOut)
		defer syscall.CloseHandle(ptyOut)

		cmdline, err := windows.UTF16PtrFromString(cmd)
		if err != nil {
			c <- err
			return
		}

		var procInfo windows.ProcessInformation
		wc.hpc, err = syscalls.CreatePseudoConsole(ptyIn, ptyOut)
		if err == nil {
			defer syscalls.ClosePseudoConsole(wc.hpc)

			var procThreadAttributeSize uintptr
			if err = syscalls.InitializeProcThreadAttributeList(nil, 1, 0, &procThreadAttributeSize); err != nil && err != windows.E_NOT_SUFFICIENT_BUFFER {
				log.Errorf("InitializeProcThreadAttributeList - first call failed: %v\n", err)
				c <- err
				return
			}
			var procHeap windows.Handle
			procHeap, err = syscalls.GetProcessHeap()
			if err != nil {
				log.Errorf("GetProcessHeap failed: %v\n", err)
				c <- err
				return
			}
			// Seems the "handle" returned is not really a handle that can't be closed...
			//defer syscall.CloseHandle(syscall.Handle(uintptr(procHeap)))
			attributeList, err := syscalls.HeapAlloc(procHeap, 0, procThreadAttributeSize)
			if err != nil {
				log.Errorf("HeapAlloc failed: %v\n", err)
				c <- err
				return
			}
			defer syscalls.HeapFree(procHeap, 0, attributeList)

			var startupInfo syscalls.StartupInfoEx
			startupInfo.Cb = uint32(unsafe.Sizeof(startupInfo))
			startupInfo.AttributeList = (*syscalls.PROC_THREAD_ATTRIBUTE_LIST)(unsafe.Pointer(attributeList))

			if err = syscalls.InitializeProcThreadAttributeList(startupInfo.AttributeList, 1, 0, &procThreadAttributeSize); err != nil {
				log.Errorf("InitializeProcThreadAttributeList failed: %v\n", err)
				c <- err
				return
			}

			defer syscalls.DeleteProcThreadAttributeList(startupInfo.AttributeList)
			h := uintptr(wc.hpc)
			if err = syscalls.UpdateProcThreadAttribute(startupInfo.AttributeList, 0, uintptr(syscalls.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE), h, unsafe.Sizeof(h), 0, nil); err != nil {
				log.Errorf("UpdateProcThreadAttribute failed: %v\n", err)
				c <- err
				return
			}

			// Start the new process
			//appName, err := windows.UTF16PtrFromString("C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe")

			//startupInfo.Flags |= windows.STARTF_USESHOWWINDOW
			//startupInfo.ShowWindow = windows.SW_HIDE
			// creationFlags := windows.CREATE_SUSPENDED | windows.CREATE_NO_WINDOW | windows.EXTENDED_STARTUPINFO_PRESENT
			creationFlags := windows.EXTENDED_STARTUPINFO_PRESENT //windows.CREATE_NO_WINDOW | windows.EXTENDED_STARTUPINFO_PRESENT
			psec := windows.SecurityAttributes{}
			psec.Length = uint32(unsafe.Sizeof(psec))
			tsec := windows.SecurityAttributes{}
			tsec.Length = uint32(unsafe.Sizeof(tsec))
			if err = syscalls.CreateProcess(nil, cmdline, &psec, &tsec, false, uint32(creationFlags), nil, nil, &startupInfo, &procInfo); err != nil {
				log.Errorf("CreateProcess failed: %v\n", err)
				c <- err
				return
			}
		} else {
			//conn.WriteMessage(websocket.TextMessage, []byte("PseudoConsole not supported on this Windows build... you are getting the raw io"))

			var startupInfo syscalls.StartupInfoEx
			startupInfo.Cb = uint32(unsafe.Sizeof(startupInfo))

			startupInfo.StdInput = windows.Handle(ptyIn)
			startupInfo.StdOutput = windows.Handle(ptyOut)
			startupInfo.StdErr = windows.Handle(ptyOut)
			startupInfo.Flags = 0x100
			//creationFlags := windows.CREATE_NO_WINDOW

			if err = syscalls.CreateProcess(nil, cmdline, nil, nil, true, 0, nil, nil, &startupInfo, &procInfo); err != nil {
				log.Errorf("CreateProcess failed: %v\n", err)
				c <- err
				return
			}
		}

		wc.procInfo = procInfo
		c <- nil

		go io.Copy(wc.conn, &syscalls.HandleConn{cmdOut})
		go io.Copy(&syscalls.HandleConn{cmdIn}, wc.conn)

		syscall.WaitForSingleObject(syscall.Handle(procInfo.Process), syscall.INFINITE)
		syscall.CloseHandle(syscall.Handle(procInfo.Thread))
		syscall.CloseHandle(syscall.Handle(procInfo.Process))

		wc.fchan <- true
	}()

	err = <-c
	if err != nil {

	}

	return &wc, err
}

func (wc *WinConsole) Close() error {
	return syscall.TerminateProcess(syscall.Handle(wc.procInfo.Process), 0)
}

func (wc *WinConsole) Resize(cols uint16, rows uint16) error {
	coord := &syscalls.COORD{X: int16(cols), Y: int16(rows)}
	err := syscalls.ResizePseudoConsole(wc.hpc, coord)
	if err != nil {
		return err
	}

	return nil
}

func (ws *WinConsole) Wait() {
	<-ws.fchan
}
