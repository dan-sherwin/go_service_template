package rpc

import (
	"fmt"
	"log/slog"
	"net"
	"net/rpc"
	"os"
	"path/filepath"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"

	"go.corp.spacelink.com/sdks/go/utilities"
)

var (
	listener   net.Listener
	SocketPath string
)

func DefaultSocketPath() string {
	if p := os.Getenv("RPC_SOCKET_PATH"); p != "" {
		return p
	}
	if r := os.Getenv("XDG_RUNTIME_DIR"); r != "" {
		return filepath.Join(r, consts.APPNAME, consts.APPNAME+"-rpc.sock")
	}
	return filepath.Join(os.TempDir(), consts.APPNAME+"-rpc.sock")
}

func init() {
	SocketPath = DefaultSocketPath()
}

func Register(rcvr any) {
	err := rpc.Register(rcvr)
	if err != nil {
		slog.Error("Register error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func RegisterName(name string, rcvr any) {
	err := rpc.RegisterName(name, rcvr)
	if err != nil {
		slog.Error("Register error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func Shutdown() {
	_ = listener.Close()
	_ = os.Remove(SocketPath)
}

func StartServer() error {
	_ = os.Remove(SocketPath)
	if err := os.MkdirAll(filepath.Dir(SocketPath), 0o770); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	var err error
	listener, err = net.Listen("unix", SocketPath)
	if err != nil {
		slog.Error("RPC listen failed", slog.String("socket", SocketPath), slog.String("error", err.Error()))
		os.Exit(1)
	}
	_ = os.Chmod(SocketPath, 0o660)
	slog.Info("RPC server listening", slog.String("socket", SocketPath))
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if _, ok := err.(*net.OpError); ok {
					slog.Info("RPC listener closing")
					return
				}
				slog.Error("RPC accept error", slog.String("error", err.Error()))
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()
	return nil
}

func Client() *rpc.Client {
	if _, err := utilities.FindDaemonProcessPID(consts.APPNAME); err != nil {
		fmt.Printf("RPC comms error.  Unable to find daemon process.  Is the daemon running? (%s)", err.Error())
		slog.Error("RPC comms error.  Unable to find daemon process.  Is the daemon running?", slog.String("error", err.Error()))
		os.Exit(1)
	}

	client, err := rpc.Dial("unix", SocketPath)
	if err != nil {
		fmt.Printf("RPC comms error.  Unable to establish comms link.  Is the daemon running? (%s)", err.Error())
		slog.Error("RPC comms error. Unable to establish comms link. Is the daemon running?", slog.String("error", err.Error()))
		os.Exit(1)
	}
	return client
}

func Call(serviceMethod string, args any, reply any) error {
	if args == nil {
		args = &struct{}{}
	}
	if _, err := utilities.FindDaemonProcessPID(consts.APPNAME); err != nil {
		msg := "RPC comms error. Daemon not found"
		fmt.Println(msg+": ", err)
		slog.Error(msg, slog.String("error", err.Error()))
		os.Exit(1)
	}
	client, err := rpc.Dial("unix", SocketPath)
	if err != nil {
		msg := "RPC comms error. Dial failed"
		fmt.Println(msg+": ", err)
		slog.Error(msg, slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer client.Close()
	return client.Call(serviceMethod, args, reply)
}
