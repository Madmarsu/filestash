package model

import (
	"os"
	"context"
	. "github.com/mickael-kerjean/filestash/server/common"
	"github.com/mickael-kerjean/net/webdav"
	"path/filepath"
	"strings"
	"io"
)

const DAVCachePath = "data/cache/webdav/"
var cachePath string

func init() {
	cachePath = filepath.Join(GetCurrentDir(), DAVCachePath) + "/"
	os.RemoveAll(cachePath)
	os.MkdirAll(cachePath, os.ModePerm)
}

type WebdavFs struct {
	backend IBackend
	path string
}

func NewWebdavFs(b IBackend, path string) WebdavFs {
	return WebdavFs{
		backend: b,
		path: path,
	}
}

func (fs WebdavFs) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	Log.Info("MKDIR ('%s')", name)
	if name = fs.resolve(name); name == "" {
		return os.ErrInvalid
	}
	return fs.backend.Mkdir(name)
}

func (fs WebdavFs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	Log.Info("OPEN_FILE ('%s')", name)
	return NewWebdavNode(name, fs), nil
}

func (fs WebdavFs) RemoveAll(ctx context.Context, name string) error {
	Log.Info("RM ('%s')", name)
	if name = fs.resolve(name); name == "" {
		return os.ErrInvalid
	}
	return fs.backend.Rm(name)
}

func (fs WebdavFs) Rename(ctx context.Context, oldName, newName string) error {
	Log.Info("MV ('%s' => '%s')", oldName, newName)
	if oldName = fs.resolve(oldName); oldName == "" {
		return os.ErrInvalid
	}
	if newName = fs.resolve(newName); newName == "" {
		return os.ErrInvalid
	}
	return fs.backend.Mv(oldName, newName)
}

func (fs WebdavFs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	Log.Info("STAT ('%s')", name)
	if name = fs.resolve(name); name == "" {
		return nil, os.ErrInvalid
	}

	if obj, ok := fs.backend.(interface{ Stat(path string) (os.FileInfo, error) }); ok {
		return obj.Stat(name)
	}
	return nil, os.ErrInvalid
}

func (fs WebdavFs) resolve(path string) string {
	p := filepath.Join(fs.path, path)
	if strings.HasSuffix(path, "/") == true && strings.HasSuffix(p, "/") == false {
		p += "/"
	}
	if strings.HasPrefix(p, fs.path) == true {
		return p
	}
	return ""
}


type WebdavNode struct {
	fs        WebdavFs
	path      string
	fileread  *os.File
	filewrite *os.File
}

func NewWebdavNode(name string, fs WebdavFs) *WebdavNode {
	return &WebdavNode{
		fs: fs,
		path: name,
	}
}

func (w *WebdavNode) Readdir(count int) ([]os.FileInfo, error) {
	Log.Info("  => READ_DIR ('%s')", w.path)
	var path string
	if path = w.fs.resolve(w.path); path == "" {
		return nil, os.ErrInvalid
	}
	return w.fs.backend.Ls(path)
}

func (w *WebdavNode) Stat() (os.FileInfo, error) {
	Log.Info("  => STAT ('%s')", w.path)
	// if w.filewrite != nil {
	// 	var path stringc
	// 	var err error

	// 	if path = w.fs.resolve(w.path); path == "" {
	// 		return nil, os.ErrInvalid
	// 	}
	// 	name := w.filewrite.Name()
	// 	w.filewrite.Close()
	// 	if w.filewrite, err = os.OpenFile(name, os.O_RDONLY, os.ModePerm); err != nil {
	// 		return nil, os.ErrInvalid
	// 	}

	// 	if err = w.fs.backend.Save(path, w.filewrite); err != nil {
	// 		return nil, err
	// 	}
	// }
	return w.fs.Stat(context.Background(), w.path)
}

func (w *WebdavNode) Close() error {
	Log.Info("  => CLOSE ('%s')", w.path)
	if w.fileread != nil {
		if err := w.cleanup(w.fileread); err != nil {
			return err
		}
		w.fileread = nil
	}
	if w.filewrite != nil {
		defer w.cleanup(w.filewrite)
		name := w.filewrite.Name()
		w.filewrite.Close()
		reader, err := os.OpenFile(name, os.O_RDONLY, os.ModePerm);
		if err != nil {
			return os.ErrInvalid
		}
		path := w.fs.resolve(w.path)
		if path == "" {
			return os.ErrInvalid
		}
		if err := w.fs.backend.Save(path, reader); err != nil {
			return err
		}
		reader.Close()
	}
	return nil
}

func (w *WebdavNode) Read(p []byte) (int, error) {
	Log.Info("  => READ ('%s')", w.path)
	if w.fileread != nil {
		return w.fileread.Read(p)
	}
	return -1, os.ErrInvalid
}

func (w *WebdavNode) Seek(offset int64, whence int) (int64, error) {
	Log.Info("  => SEEK ('%s')", w.path)
	var path string
	var err error
	if path = w.fs.resolve(w.path); path == "" {
		return 0, os.ErrInvalid
	}

	if w.fileread == nil {
		var reader io.Reader
		if w.fileread, err = os.OpenFile(cachePath + "tmp_" + QuickString(10), os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.ModePerm); err != nil {
			return 0, os.ErrInvalid
		}
		if reader, err = w.fs.backend.Cat(path); err != nil {
			return 0, os.ErrInvalid
		}
		io.Copy(w.fileread, reader)

		name := w.fileread.Name()
		w.fileread.Close()
		w.fileread, err = os.OpenFile(name, os.O_RDONLY, os.ModePerm)
	}
	return w.fileread.Seek(offset, whence)
}

func (w *WebdavNode) Write(p []byte) (int, error) {
	Log.Info("  => WRITE ('%s')", w.path)
	var err error

	if w.filewrite == nil {
		if w.filewrite, err = os.OpenFile(cachePath + "tmp_" + QuickString(10), os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.ModePerm); err != nil {
			return 0, os.ErrInvalid
		}
	}
	return w.filewrite.Write(p)
}

func (w *WebdavNode) cleanup(file *os.File) error {
	name := file.Name()
	file.Close();
	os.Remove(name);
	return nil
}
