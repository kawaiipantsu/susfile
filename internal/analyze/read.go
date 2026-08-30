package analyze

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

// input abstracts "the bytes to analyse", whether they come from a regular file
// (seekable, re-readable) or from standard input (read once into memory).
type input struct {
	displayPath string
	name        string
	size        int64
	mode        fs.FileMode
	modTime     time.Time
	fromStdin   bool

	f   *os.File // nil for stdin
	mem []byte   // whole stdin content, or nil
}

func openInput(path string, opt Options) (*input, error) {
	if path == "-" {
		return openStdin(opt)
	}
	return openFile(path, opt)
}

func openStdin(opt Options) (*input, error) {
	// Bound stdin so a hostile pipe cannot exhaust memory. We keep one extra
	// byte to notice (and warn about) truncation later.
	lim := opt.MaxBytes
	data, err := io.ReadAll(io.LimitReader(os.Stdin, lim+1))
	if err != nil {
		return nil, fmt.Errorf("%w: reading stdin: %v", ErrNotReadable, err)
	}
	return &input{
		displayPath: "<stdin>",
		name:        "<stdin>",
		size:        int64(len(data)),
		fromStdin:   true,
		mem:         data,
	}, nil
}

func openFile(path string, opt Options) (*input, error) {
	li, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotReadable, err)
	}
	mode := li.Mode()

	// Resolve a symlink to its target for the type check, but keep the name
	// the user gave us.
	st := li
	if mode&fs.ModeSymlink != 0 {
		st, err = os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("%w: broken symlink: %v", ErrNotReadable, err)
		}
		mode = st.Mode()
	}

	switch {
	case mode.IsDir():
		return nil, fmt.Errorf("%w: %s is a directory", ErrNotReadable, path)
	case mode.IsRegular():
		// ok
	case !opt.AllowSpecial:
		return nil, fmt.Errorf("%w: %s is not a regular file (use --allow-special)", ErrNotReadable, path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotReadable, err)
	}

	size := st.Size()
	if !mode.IsRegular() {
		// Devices and FIFOs report size 0; we will discover the real length by
		// reading. Leave size at 0 so progress reporting stays sane.
		size = 0
	}

	return &input{
		displayPath: path,
		name:        baseName(path),
		size:        size,
		mode:        mode,
		modTime:     st.ModTime(),
		f:           f,
	}, nil
}

// reader returns a fresh reader positioned at the start of the input.
func (in *input) reader() io.Reader {
	if in.fromStdin {
		return bytes.NewReader(in.mem)
	}
	_, _ = in.f.Seek(0, io.SeekStart)
	return in.f
}

// bufferPrefix returns up to max bytes from the start of the input, and whether
// more data followed (truncation).
func (in *input) bufferPrefix(max int64) (buf []byte, truncated bool, err error) {
	if in.fromStdin {
		if int64(len(in.mem)) > max {
			return in.mem[:max], true, nil
		}
		return in.mem, false, nil
	}

	if _, err = in.f.Seek(0, io.SeekStart); err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrNotReadable, err)
	}
	buf, err = io.ReadAll(io.LimitReader(in.f, max))
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrNotReadable, err)
	}
	// One more byte tells us whether the file continued past the cap.
	var probe [1]byte
	n, _ := in.f.Read(probe[:])
	return buf, n > 0, nil
}

// readAt fills p from off; used by the sampling micro-block scan. Never called
// for stdin (which is not seekable beyond its buffer).
func (in *input) readAt(p []byte, off int64) (int, error) {
	if in.f == nil {
		return 0, io.ErrUnexpectedEOF
	}
	return in.f.ReadAt(p, off)
}

func (in *input) Close() error {
	if in.f != nil {
		return in.f.Close()
	}
	return nil
}
