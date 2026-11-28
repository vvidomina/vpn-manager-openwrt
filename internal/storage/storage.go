// Package storage — атомарная запись и чтение файлов проекта.
// Схема: временный файл в том же каталоге → fsync → rename → fsync каталога.
// Системные файлы OpenWrt не изменяются.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vvidomina/vpn-manager-openwrt/internal/vmerrors"
)

// Store — корневой каталог данных приложения.
type Store struct {
	root string
}

// New создаёт хранилище для каталога root.
func New(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, vmerrors.New(vmerrors.CodeInvalidArgument, "storage.new", "пустой корневой каталог")
	}
	return &Store{root: root}, nil
}

// Root возвращает корневой каталог.
func (s *Store) Root() string { return s.root }

// Path возвращает путь внутри корня; выход за пределы корня запрещён.
func (s *Store) Path(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", vmerrors.New(vmerrors.CodeInvalidArgument, "storage.path", "пустое имя файла")
	}
	// filepath.IsAbs на Windows не считает абсолютным путь вида /etc/passwd,
	// поэтому проверяются также разделители в начале и имя тома.
	if filepath.IsAbs(name) || filepath.VolumeName(name) != "" ||
		strings.HasPrefix(name, "/") || strings.HasPrefix(name, `\`) ||
		strings.Contains(name, "..") {
		return "", vmerrors.New(vmerrors.CodeInvalidArgument, "storage.path", "недопустимое имя файла: "+name)
	}
	return filepath.Join(s.root, name), nil
}

// Read читает файл. Отсутствие файла — VM_ERR_NOT_FOUND.
func (s *Store) Read(name string) ([]byte, error) {
	path, err := s.Path(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, vmerrors.Wrap(vmerrors.CodeNotFound, "storage.read", err)
		}
		return nil, vmerrors.Wrap(vmerrors.CodeStorage, "storage.read", err)
	}
	return data, nil
}

// WriteAtomic записывает данные атомарно с правами perm.
func (s *Store) WriteAtomic(name string, data []byte, perm os.FileMode) error {
	path, err := s.Path(name)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return vmerrors.Wrap(vmerrors.CodeStorage, "storage.write", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return vmerrors.Wrap(vmerrors.CodeStorage, "storage.write.tmp", err)
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return vmerrors.Wrap(vmerrors.CodeStorage, "storage.write.data", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return vmerrors.Wrap(vmerrors.CodeStorage, "storage.write.sync", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return vmerrors.Wrap(vmerrors.CodeStorage, "storage.write.close", err)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		_ = os.Remove(tmpName)
		return vmerrors.Wrap(vmerrors.CodeStorage, "storage.write.chmod", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return vmerrors.Wrap(vmerrors.CodeStorage, "storage.write.rename", err)
	}

	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// String реализует fmt.Stringer для диагностики.
func (s *Store) String() string { return fmt.Sprintf("storage{root=%s}", s.root) }
