package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

func TestWriteAndRead(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.WriteAtomic("state/snapshot.json", []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatalf("запись: %v", err)
	}
	got, err := st.Read("state/snapshot.json")
	if err != nil {
		t.Fatalf("чтение: %v", err)
	}
	if string(got) != `{"ok":true}` {
		t.Fatalf("содержимое = %q", got)
	}
	info, err := os.Stat(filepath.Join(st.Root(), "state", "snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Windows не хранит POSIX-права: проверка имеет смысл только на Unix.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("права = %v, ожидались 0600", info.Mode().Perm())
	}
}

func TestReadMissing(t *testing.T) {
	st, _ := New(t.TempDir())
	if _, err := st.Read("нет.json"); !vmerrors.Is(err, vmerrors.CodeNotFound) {
		t.Fatalf("ожидался VM_ERR_NOT_FOUND, получено %v", err)
	}
}

func TestPathTraversal(t *testing.T) {
	st, _ := New(t.TempDir())
	for _, name := range []string{"../secret", "/etc/passwd", "", "  "} {
		if _, err := st.Path(name); !vmerrors.Is(err, vmerrors.CodeInvalidArgument) {
			t.Fatalf("путь %q должен быть отклонён, получено %v", name, err)
		}
	}
}

func TestNewEmptyRoot(t *testing.T) {
	if _, err := New(" "); !vmerrors.Is(err, vmerrors.CodeInvalidArgument) {
		t.Fatalf("ожидался VM_ERR_INVALID_ARGUMENT, получено %v", err)
	}
}
