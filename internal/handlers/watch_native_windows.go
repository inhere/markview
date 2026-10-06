//go:build windows

package handlers

import (
	"errors"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/gookit/goutil/x/clog"
	"github.com/inhere/markview/internal/config"
	"golang.org/x/sys/windows"
)

var errWatcherClosed = errors.New("watcher closed")

// notifyFilter 与 fsnotify 的 Windows 后端保持一致。
const notifyFilter uint32 = windows.FILE_NOTIFY_CHANGE_FILE_NAME |
	windows.FILE_NOTIFY_CHANGE_DIR_NAME |
	windows.FILE_NOTIFY_CHANGE_LAST_WRITE |
	windows.FILE_NOTIFY_CHANGE_SIZE

// winWatcher 用单个 ReadDirectoryChangesW 句柄递归监听整棵项目树。
// 因为不再对每个子目录单独打开句柄，所以运行期间目录仍可正常移动/重命名。
type winWatcher struct {
	root    ProjectRoot
	cfg     config.Config
	handle  windows.Handle
	event   windows.Handle
	ov      windows.Overlapped
	buf     []byte
	changes chan nativeChange
	errs    chan error
	done    chan struct{}
	once    sync.Once

	mu     sync.Mutex
	closed bool
	armed  bool
}

func newNativeWatcher(root ProjectRoot, cfg config.Config) (nativeWatcher, error) {
	pathPtr, err := windows.UTF16PtrFromString(root.RealPath)
	if err != nil {
		return nil, err
	}
	// 打开目录句柄需要 FILE_FLAG_BACKUP_SEMANTICS；异步读取需要 FILE_FLAG_OVERLAPPED。
	handle, err := windows.CreateFile(pathPtr,
		windows.FILE_LIST_DIRECTORY,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil, err
	}
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		windows.CloseHandle(handle)
		return nil, err
	}

	watcher := &winWatcher{
		root:    root,
		cfg:     cfg,
		handle:  handle,
		event:   event,
		buf:     make([]byte, 64*1024),
		changes: make(chan nativeChange, 32),
		errs:    make(chan error, 4),
		done:    make(chan struct{}),
	}
	watcher.ov.HEvent = event
	go watcher.loop()
	return watcher, nil
}

func (w *winWatcher) Events() <-chan nativeChange { return w.changes }
func (w *winWatcher) Errors() <-chan error        { return w.errs }

func (w *winWatcher) Close() error {
	w.once.Do(func() {
		w.mu.Lock()
		w.closed = true
		armed := w.armed
		w.mu.Unlock()

		close(w.done)
		// 取消挂起的读取，唤醒 loop 退出。
		if armed {
			_ = windows.CancelIoEx(w.handle, &w.ov)
		}
		_ = windows.CloseHandle(w.handle)
		_ = windows.CloseHandle(w.event)
	})
	return nil
}

// arm 发起一次挂起的目录变更读取。watchSubTree 传入 true 使单个句柄覆盖整棵子树。
func (w *winWatcher) arm() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return errWatcherClosed
	}
	err := windows.ReadDirectoryChanges(w.handle, &w.buf[0], uint32(len(w.buf)),
		true, notifyFilter, nil, &w.ov, 0)
	if err != nil && err != windows.ERROR_IO_PENDING {
		return err
	}
	w.armed = true
	return nil
}

func (w *winWatcher) loop() {
	defer close(w.changes)
	defer close(w.errs)

	for {
		if err := w.arm(); err != nil {
			if !errors.Is(err, errWatcherClosed) {
				w.reportError(err)
			}
			return
		}

		result, err := windows.WaitForSingleObject(w.event, windows.INFINITE)
		if err != nil || result != windows.WAIT_OBJECT_0 {
			if err != nil && !w.isClosed() {
				w.reportError(err)
			}
			return
		}

		var size uint32
		if err := windows.GetOverlappedResult(w.handle, &w.ov, &size, false); err != nil {
			// Close 时 CancelIoEx 会得到 ERROR_OPERATION_ABORTED。
			if !w.isClosed() {
				w.reportError(err)
			}
			return
		}

		if size == 0 {
			// 一次变更过多导致缓冲区溢出，内核会丢弃事件。
			clog.Warnf("WATCH: change buffer overflowed for %s, some changes may be missed", w.root.DisplayPath)
			continue
		}
		w.parse(w.buf[:size])
	}
}

func (w *winWatcher) parse(buf []byte) {
	for {
		info := (*windows.FileNotifyInformation)(unsafe.Pointer(&buf[0]))
		units := unsafe.Slice(&info.FileName, int(info.FileNameLength)/2)
		w.handleNotify(info.Action, windows.UTF16ToString(units))

		if info.NextEntryOffset == 0 {
			return
		}
		buf = buf[info.NextEntryOffset:]
	}
}

func (w *winWatcher) handleNotify(action uint32, name string) {
	rel := filepath.ToSlash(name)
	switch action {
	case windows.FILE_ACTION_ADDED, windows.FILE_ACTION_RENAMED_NEW_NAME:
		w.emit(rel, EventTypeCreate)
	case windows.FILE_ACTION_MODIFIED:
		w.emit(rel, EventTypeUpdate)
	}
}

func (w *winWatcher) emit(rel, op string) {
	change, ok := markdownChangeFromRel(rel, op, w.cfg)
	if !ok {
		return
	}
	select {
	case w.changes <- change:
	case <-w.done:
	}
}

func (w *winWatcher) reportError(err error) {
	select {
	case w.errs <- err:
	default:
	}
}

func (w *winWatcher) isClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}
