package sys

import (
	"io"
	"io/fs"
	"net"
	"syscall"

	"github.com/tetratelabs/wazero/internal/descriptor"
	"github.com/tetratelabs/wazero/internal/fsapi"
	socketapi "github.com/tetratelabs/wazero/internal/sock"
	"github.com/tetratelabs/wazero/internal/sysfs"
)

const (
	FdStdin int32 = iota
	FdStdout
	FdStderr
	// FdPreopen is the file descriptor of the first pre-opened directory.
	//
	// # Why file descriptor 3?
	//
	// While not specified, the most common WASI implementation, wasi-libc,
	// expects POSIX style file descriptor allocation, where the lowest
	// available number is used to open the next file. Since 1 and 2 are taken
	// by stdout and stderr, the next is 3.
	//   - https://github.com/WebAssembly/WASI/issues/122
	//   - https://pubs.opengroup.org/onlinepubs/9699919799/functions/V2_chap02.html#tag_15_14
	//   - https://github.com/WebAssembly/wasi-libc/blob/wasi-sdk-16/libc-bottom-half/sources/preopens.c#L215
	FdPreopen
)

const modeDevice = fs.ModeDevice | 0o640

// FileEntry maps a path to an open file in a file system.
type FileEntry struct {
	// Name is the name of the directory up to its pre-open, or the pre-open
	// name itself when IsPreopen.
	//
	// # Notes
	//
	//   - This can drift on rename.
	//   - This relates to the guest path, which is not the real file path
	//     except if the entire host filesystem was made available.
	Name string

	// IsPreopen is a directory that is lazily opened.
	IsPreopen bool

	// FS is the filesystem associated with the pre-open.
	FS fsapi.FS

	// File is always non-nil.
	File fsapi.File

	// openDir is nil until Opendir was called.
	openDir *Dir
}

// Opendir opens a directory stream associated with file. The Dir result is
// stateful, so subsequent calls return any next values in directory order.
//
// # Parameters
//
// When `addDotEntries` is true, navigational entries "." and ".." precede
// any other entries in the directory. Otherwise, they will be absent.
//
// # Errors
//
// A zero syscall.Errno is success. The below are expected otherwise:
//   - syscall.ENOSYS: the implementation does not support this function.
//   - syscall.EBADF: the dir was closed or not readable.
//   - syscall.ENOTDIR: the file was not a directory.
//
// # Notes
//
//   - This is like `opendir` in POSIX.
//     See https://pubs.opengroup.org/onlinepubs/9699919799/functions/opendir.html
//   - `addDotEntries` is only needed to pass wasi-testsuite for
//     "wasi_snapshot_preview1.fd_readdir". Using this has numerous
//     downsides as detailed in /RATIONALE.md
func (f *FileEntry) Opendir(addDotEntries bool) (dir *Dir, errno syscall.Errno) {
	if dir = f.openDir; dir != nil {
		return dir, 0
	} else if dir, errno = newReaddirFromFileEntry(f, addDotEntries); errno != 0 {
		// not a directory or error reading it.
		return nil, errno
	} else {
		f.openDir = dir
		return dir, 0
	}
}

const direntBufSize = 16

// Dir is an open directory stream, created by FileEntry.Opendir.
//
// # Notes
//
//   - This is similar to DIR in POSIX. See
//     https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/dirent.h.html
//   - Tell and Seek are not implemented with system calls, so do not return
//     syscall.Errno.
//   - Implementations should consider implementing this with bulk reads, like
//     `getdents` in Linux. This avoids excessive syscalls when iterating a
//     large directory.
//   - `wasi_snapshot_preview1.fd_readdir` is the primary caller of Tell and
//     Seek. To implement pagination similar to `getdents` on Linux, the value
//     of Tell is used for `dirent.d_next` and Seek is used to continue at that
//     position, or a prior in the case of truncation.
//   - You must call Close to avoid file resource conflicts. For example,
//     Windows cannot delete the underlying directory while a handle to it
//     remains open.
type Dir struct {
	// pos is the current position in the buffer.
	pos uint64

	// countRead is the total count of files read including Dirents.
	//
	// Notes:
	//
	// * countRead is the index of the next file in the list. This is
	//   also the value that Tell returns, so it should always be
	//   higher or equal than the pos given in Seek.
	//
	// * this can overflow to negative, which means our implementation
	//   doesn't support writing greater than max int64 entries.
	//   countRead uint64
	countRead uint64

	// dirents is a fixed buffer of size direntBufSize. Notably,
	// directory listing are not rewindable, so we keep entries around in case
	// the caller mis-estimated their buffer and needs a few still cached.
	//
	// Note: This is wasi-specific and needs to be refactored.
	// In wasi preview1, dot and dot-dot entries are required to exist, but the
	// reverse is true for preview2. More importantly, preview2 holds separate
	// stateful dir-entry-streams per file.
	dirents []fsapi.Dirent

	// dirInit seeks and reset the provider for dirents to the beginning
	// and returns an initial batch (e.g. dot directories).
	dirInit func() ([]fsapi.Dirent, syscall.Errno)

	// dirReader fetches a new batch of direntBufSize elements.
	dirReader func(n uint64) ([]fsapi.Dirent, syscall.Errno)
}

func NewReaddir(
	dirInit func() ([]fsapi.Dirent, syscall.Errno),
	dirReader func(n uint64) ([]fsapi.Dirent, syscall.Errno),
) (*Dir, syscall.Errno) {
	d := &Dir{dirReader: dirReader, dirInit: dirInit}
	return d, d.init()
}

// init resets the pos and invokes the dirInit, dirReader
// methods to reset the internal state of the Readdir struct.
//
// Note: this is different from Reset, because it will not short-circuit
// when pos is already 0, but it will force an unconditional reload.
func (d *Dir) init() syscall.Errno {
	d.pos = 0
	d.countRead = 0
	// Reset the buffer to the initial state.
	initialDirents, errno := d.dirInit()
	if errno != 0 {
		return errno
	}
	if len(initialDirents) > direntBufSize {
		return syscall.EINVAL
	}
	d.dirents = initialDirents
	// Fill the buffer with more data.
	count := direntBufSize - len(initialDirents)
	if count == 0 {
		// No need to fill up the buffer further.
		return 0
	}
	dirents, errno := d.dirReader(uint64(count))
	if errno != 0 {
		return errno
	}
	d.dirents = append(d.dirents, dirents...)
	return 0
}

// newReaddirFromFileEntry is a constructor for Readdir that takes a FileEntry to initialize.
func newReaddirFromFileEntry(f *FileEntry, addDotEntries bool) (*Dir, syscall.Errno) {
	var dotEntries []fsapi.Dirent
	if addDotEntries {
		// Generate the dotEntries only once and return it many times in the dirInit closure.
		var errno syscall.Errno
		if dotEntries, errno = synthesizeDotEntries(f); errno != 0 {
			return nil, errno
		}
	}
	dirInit := func() ([]fsapi.Dirent, syscall.Errno) {
		// Ensure we always rewind to the beginning when we re-init.
		if _, errno := f.File.Seek(0, io.SeekStart); errno != 0 {
			return nil, errno
		}
		// Return the dotEntries that we have already generated outside the closure.
		return dotEntries, 0
	}
	dirReader := func(n uint64) ([]fsapi.Dirent, syscall.Errno) { return f.File.Readdir(int(n)) }
	return NewReaddir(dirInit, dirReader)
}

// synthesizeDotEntries generates a slice of the two elements "." and "..".
func synthesizeDotEntries(f *FileEntry) (result []fsapi.Dirent, errno syscall.Errno) {
	dotIno, errno := f.File.Ino()
	if errno != 0 {
		return nil, errno
	}
	result = append(result, fsapi.Dirent{Name: ".", Ino: dotIno, Type: fs.ModeDir})
	// See /RATIONALE.md for why we don't attempt to get an inode for ".."
	result = append(result, fsapi.Dirent{Name: "..", Ino: 0, Type: fs.ModeDir})
	return result, 0
}

// Reset seeks the internal pos to 0 and refills the buffer.
func (d *Dir) Reset() syscall.Errno {
	if d.countRead == 0 {
		return 0
	}
	return d.init()
}

// Skip is equivalent to calling n times Advance.
func (d *Dir) Skip(n uint64) {
	end := d.countRead + n
	var err syscall.Errno = 0
	for d.countRead < end && err == 0 {
		err = d.Advance()
	}
}

// Tell returns the current position in the directory stream.
//
// This only has meaning if called after a successful call to Read. If
// Read returned false, this value is undefined, but conventionally zero.
//
// # Errors
//
// This operation is not implemented with system calls, so does not return
// an error.
//
// # Notes
//
//   - This is similar `telldir` in POSIX. See
//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/seekdir.html
//   - This value should not be interpreted as a number because the
//     implementation might not be backed by a numeric index.
//   - Do not confuse this with `linux_dirent.d_off` from `getdents`: the
//     location of the next entry. This is the location of the current one.
//     See https://man7.org/linux/man-pages/man2/getdents.2.html
func (d *Dir) Tell() uint64 {
	return d.pos
}

// Seek sets the position for the next call to Read.
//
// When `loc == 0`, the directory will be reset to its initial state.
// Otherwise, `loc` should be a former value returned by Tell.
//
// # Errors
//
// This operation is not implemented with system calls, so does not return
// an error, even if `loc` is invalid. An invalid `loc` results in the next
// Read returning syscall.ENOENT.
//
// # Notes
//
//   - This is similar `seekdir` in POSIX. See
//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/seekdir.html
//   - A zero value is similar to calling `rewinddir` in POSIX. See
//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/rewinddir.html
//   - `loc == 0` can be implemented by setting a flag that re-opens the
//     underlying directory and dumps any cache on the next call to Read.
//   - `loc != 0` can be implemented with cached dirents returned by Read,
//     kept in a sliding window. The sliding window avoids out of memory
//     errors reading large directories. If loc is not in the window, the
//     next call to Read would fail with syscall.ENOENT.
func (d *Dir) Seek(loc uint64) syscall.Errno {
	switch {
	case loc > d.countRead:
		// the pos can neither be negative nor can it be larger than countRead.
		return syscall.ENOENT
	case loc == 0 && d.countRead == 0:
		return 0
	case loc == 0 && d.countRead != 0:
		// This means that there was a previous call to the dir, but pos is reset.
		// This happens when the program calls rewinddir, for example:
		// https://github.com/WebAssembly/wasi-libc/blob/659ff414560721b1660a19685110e484a081c3d4/libc-bottom-half/cloudlibc/src/libc/dirent/rewinddir.c#L10-L12
		return d.Reset()
	case loc < d.countRead:
		if loc/direntBufSize != uint64(d.countRead)/direntBufSize {
			// The pos is not 0, but it points into a window before the current one.
			return syscall.ENOENT
		}
		// We are allowed to rewind back to a previous offset within the current window.
		d.countRead = loc
		d.pos = d.countRead % direntBufSize
		return 0
	default:
		// The loc is valid.
		return 0
	}
}

// Peek emits the current value.
// It returns syscall.ENOENT when there are no entries left in the directory.
func (d *Dir) Peek() (*fsapi.Dirent, syscall.Errno) {
	switch {
	case d.pos == uint64(len(d.dirents)):
		// We're past the buf size, fill it up again.
		dirents, errno := d.dirReader(direntBufSize)
		if errno != 0 {
			return nil, errno
		}
		d.dirents = append(d.dirents, dirents...)
		fallthrough
	default: // d.pos < direntBufSize FIXME
		if d.pos == uint64(len(d.dirents)) {
			return nil, syscall.ENOENT
		}
		dirent := &d.dirents[d.pos]
		return dirent, 0
	}
}

// Advance advances the internal counters and indices to the next value.
// It also empties and refill the buffer with the next set of values when the internal pos
// reaches the end of it.
func (d *Dir) Advance() syscall.Errno {
	if d.pos == uint64(len(d.dirents)) {
		return syscall.ENOENT
	}
	d.pos++
	d.countRead++
	return 0
}

type FSContext struct {
	// rootFS is the root ("/") mount.
	rootFS fsapi.FS

	// openedFiles is a map of file descriptor numbers (>=FdPreopen) to open files
	// (or directories) and defaults to empty.
	// TODO: This is unguarded, so not goroutine-safe!
	openedFiles FileTable
}

// FileTable is a specialization of the descriptor.Table type used to map file
// descriptors to file entries.
type FileTable = descriptor.Table[int32, *FileEntry]

// RootFS returns a possibly unimplemented root filesystem. Any files that
// should be added to the table should be inserted via InsertFile.
//
// TODO: This is only used by GOOS=js and tests: Remove when we remove GOOS=js
// (after Go 1.22 is released).
func (c *FSContext) RootFS() fsapi.FS {
	if rootFS := c.rootFS; rootFS == nil {
		return fsapi.UnimplementedFS{}
	} else {
		return rootFS
	}
}

// LookupFile returns a file if it is in the table.
func (c *FSContext) LookupFile(fd int32) (*FileEntry, bool) {
	return c.openedFiles.Lookup(fd)
}

// OpenFile opens the file into the table and returns its file descriptor.
// The result must be closed by CloseFile or Close.
func (c *FSContext) OpenFile(fs fsapi.FS, path string, flag int, perm fs.FileMode) (int32, syscall.Errno) {
	if f, errno := fs.OpenFile(path, flag, perm); errno != 0 {
		return 0, errno
	} else {
		fe := &FileEntry{FS: fs, File: f}
		if path == "/" || path == "." {
			fe.Name = ""
		} else {
			fe.Name = path
		}
		if newFD, ok := c.openedFiles.Insert(fe); !ok {
			return 0, syscall.EBADF
		} else {
			return newFD, 0
		}
	}
}

// Renumber assigns the file pointed by the descriptor `from` to `to`.
func (c *FSContext) Renumber(from, to int32) syscall.Errno {
	fromFile, ok := c.openedFiles.Lookup(from)
	if !ok || to < 0 {
		return syscall.EBADF
	} else if fromFile.IsPreopen {
		return syscall.ENOTSUP
	}

	// If toFile is already open, we close it to prevent windows lock issues.
	//
	// The doc is unclear and other implementations do nothing for already-opened To FDs.
	// https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_renumberfd-fd-to-fd---errno
	// https://github.com/bytecodealliance/wasmtime/blob/main/crates/wasi-common/src/snapshots/preview_1.rs#L531-L546
	if toFile, ok := c.openedFiles.Lookup(to); ok {
		if toFile.IsPreopen {
			return syscall.ENOTSUP
		}
		_ = toFile.File.Close()
	}

	c.openedFiles.Delete(from)
	if !c.openedFiles.InsertAt(fromFile, to) {
		return syscall.EBADF
	}
	return 0
}

// SockAccept accepts a socketapi.TCPConn into the file table and returns
// its file descriptor.
func (c *FSContext) SockAccept(sockFD int32, nonblock bool) (int32, syscall.Errno) {
	var sock socketapi.TCPSock
	if e, ok := c.LookupFile(sockFD); !ok || !e.IsPreopen {
		return 0, syscall.EBADF // Not a preopen
	} else if sock, ok = e.File.(socketapi.TCPSock); !ok {
		return 0, syscall.EBADF // Not a sock
	}

	var conn socketapi.TCPConn
	var errno syscall.Errno
	if conn, errno = sock.Accept(); errno != 0 {
		return 0, errno
	} else if nonblock {
		if errno = conn.SetNonblock(true); errno != 0 {
			_ = conn.Close()
			return 0, errno
		}
	}

	fe := &FileEntry{File: conn}
	if newFD, ok := c.openedFiles.Insert(fe); !ok {
		return 0, syscall.EBADF
	} else {
		return newFD, 0
	}
}

// CloseFile returns any error closing the existing file.
func (c *FSContext) CloseFile(fd int32) (errno syscall.Errno) {
	f, ok := c.openedFiles.Lookup(fd)
	if !ok {
		return syscall.EBADF
	}
	if errno = f.File.Close(); errno != 0 {
		return errno
	}
	c.openedFiles.Delete(fd)
	return errno
}

// Close implements io.Closer
func (c *FSContext) Close() (err error) {
	// Close any files opened in this context
	c.openedFiles.Range(func(fd int32, entry *FileEntry) bool {
		if errno := entry.File.Close(); errno != 0 {
			err = errno // This means err returned == the last non-nil error.
		}
		return true
	})
	// A closed FSContext cannot be reused so clear the state.
	c.openedFiles = FileTable{}
	return
}

// InitFSContext initializes a FSContext with stdio streams and optional
// pre-opened filesystems and TCP listeners.
func (c *Context) InitFSContext(
	stdin io.Reader,
	stdout, stderr io.Writer,
	fs []fsapi.FS, guestPaths []string,
	tcpListeners []*net.TCPListener,
) (err error) {
	inFile, err := stdinFileEntry(stdin)
	if err != nil {
		return err
	}
	c.fsc.openedFiles.Insert(inFile)
	outWriter, err := stdioWriterFileEntry("stdout", stdout)
	if err != nil {
		return err
	}
	c.fsc.openedFiles.Insert(outWriter)
	errWriter, err := stdioWriterFileEntry("stderr", stderr)
	if err != nil {
		return err
	}
	c.fsc.openedFiles.Insert(errWriter)

	for i, fs := range fs {
		guestPath := guestPaths[i]

		if StripPrefixesAndTrailingSlash(guestPath) == "" {
			c.fsc.rootFS = fs
		}
		c.fsc.openedFiles.Insert(&FileEntry{
			FS:        fs,
			Name:      guestPath,
			IsPreopen: true,
			File:      &lazyDir{fs: fs},
		})
	}

	for _, tl := range tcpListeners {
		c.fsc.openedFiles.Insert(&FileEntry{IsPreopen: true, File: sysfs.NewTCPListenerFile(tl)})
	}
	return nil
}

// StripPrefixesAndTrailingSlash skips any leading "./" or "/" such that the
// result index begins with another string. A result of "." coerces to the
// empty string "" because the current directory is handled by the guest.
//
// Results are the offset/len pair which is an optimization to avoid re-slicing
// overhead, as this function is called for every path operation.
//
// Note: Relative paths should be handled by the guest, as that's what knows
// what the current directory is. However, paths that escape the current
// directory e.g. "../.." have been found in `tinygo test` and this
// implementation takes care to avoid it.
func StripPrefixesAndTrailingSlash(path string) string {
	// strip trailing slashes
	pathLen := len(path)
	for ; pathLen > 0 && path[pathLen-1] == '/'; pathLen-- {
	}

	pathI := 0
loop:
	for pathI < pathLen {
		switch path[pathI] {
		case '/':
			pathI++
		case '.':
			nextI := pathI + 1
			if nextI < pathLen && path[nextI] == '/' {
				pathI = nextI + 1
			} else if nextI == pathLen {
				pathI = nextI
			} else {
				break loop
			}
		default:
			break loop
		}
	}
	return path[pathI:pathLen]
}
