package rpc

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/rpc"
	"os"
	"path/filepath"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"

	"github.com/dan-sherwin/go-utilities"
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
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			slog.Debug("RPC listener close failed", slog.String("error", err.Error()))
		}
		listener = nil
	}
	if err := os.Remove(SocketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Debug("RPC socket cleanup failed", slog.String("socket", SocketPath), slog.String("error", err.Error()))
	}
}

func StartServer() error {
	if err := os.Remove(SocketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale rpc socket %s: %w", SocketPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(SocketPath), 0o770); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	var err error
	listener, err = net.Listen("unix", SocketPath)
	if err != nil {
		return fmt.Errorf("listen on rpc socket %s: %w", SocketPath, err)
	}
	if err := os.Chmod(SocketPath, 0o660); err != nil {
		_ = listener.Close()
		listener = nil
		return fmt.Errorf("chmod rpc socket %s: %w", SocketPath, err)
	}
	slog.Info("RPC server listening", slog.String("socket", SocketPath))
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
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

func Client() (*rpc.Client, error) {
	if _, err := utilities.FindDaemonProcessPID(consts.APPNAME); err != nil {
		return nil, fmt.Errorf("find daemon process: %w", err)
	}

	client, err := rpc.Dial("unix", SocketPath)
	if err != nil {
		return nil, fmt.Errorf("dial rpc socket %s: %w", SocketPath, err)
	}
	return client, nil
}

func Call(serviceMethod string, args any, reply any) error {
	if args == nil {
		args = &struct{}{}
	}
	client, err := Client()
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			slog.Debug("Failed to close RPC client", slog.String("error", err.Error()))
		}
	}()
	return client.Call(serviceMethod, args, reply)
}
