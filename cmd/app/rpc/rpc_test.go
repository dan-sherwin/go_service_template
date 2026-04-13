package rpc

import (
	"os"
	"path/filepath"
	"testing"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/consts"
)

func TestDefaultSocketPathUsesXDGRuntimeDir(t *testing.T) {
	t.Setenv("RPC_SOCKET_PATH", "")
	t.Setenv("XDG_RUNTIME_DIR", "/tmp/chronix-runtime")

	got := DefaultSocketPath()
	want := filepath.Join("/tmp/chronix-runtime", consts.APPNAME, consts.APPNAME+"-rpc.sock")
	if got != want {
		t.Fatalf("unexpected socket path: got %q want %q", got, want)
	}
}

func TestDefaultSocketPathUsesExplicitRPCSocketPath(t *testing.T) {
	t.Setenv("RPC_SOCKET_PATH", "/tmp/custom.sock")
	t.Setenv("XDG_RUNTIME_DIR", "/tmp/ignored-runtime")

	got := DefaultSocketPath()
	if got != "/tmp/custom.sock" {
		t.Fatalf("unexpected explicit socket path: got %q", got)
	}
}

func TestShutdownWithoutListenerDoesNotPanic(t *testing.T) {
	oldListener := listener
	oldSocketPath := SocketPath
	t.Cleanup(func() {
		listener = oldListener
		SocketPath = oldSocketPath
	})

	listener = nil
	SocketPath = filepath.Join(t.TempDir(), consts.APPNAME+"-rpc.sock")

	if err := os.WriteFile(SocketPath, []byte("stale socket placeholder"), 0o600); err != nil {
		t.Fatalf("write stale socket placeholder: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("shutdown panicked: %v", r)
		}
	}()

	Shutdown()

	if _, err := os.Stat(SocketPath); !os.IsNotExist(err) {
		t.Fatalf("expected socket path to be removed, stat err=%v", err)
	}
}
