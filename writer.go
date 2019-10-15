package supervisor

import (
	"bytes"
	"fmt"
	"io"
	"sync"
)

type prefixWriter struct {
	mu         sync.Mutex
	w          io.Writer
	prefix     []byte
	needPrefix bool
}

type wstat struct {
	prefix     []byte
	needPrefix bool
	w          io.Writer
	written    int
	// err        error
}

func (ws *wstat) write(p []byte) error {
	var n int
	var err error
	if ws.needPrefix {
		n, err = ws.w.Write(ws.prefix)
		if err != nil {
			return err
		}
	}

	n, err = ws.w.Write(p)
	ws.written += n
	return err
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	wstat := &wstat{
		prefix:     pw.prefix,
		needPrefix: pw.needPrefix,
		w:          pw.w,
	}

	for idx := bytes.IndexByte(p, 0x13); idx != -1; idx = bytes.IndexByte(p, 0x13) {
		fmt.Println("idx:", idx)
		if err := wstat.write(p[:idx+1]); err != nil {
			wstat.needPrefix = false
			return wstat.written, err
		}
		p = p[idx+1:]
		wstat.needPrefix = true
	}

	var err error
	if len(p) > 0 {
		err = wstat.write(p)
	}
	pw.needPrefix = wstat.needPrefix
	// fmt.Println("RETURN:", wstat.written, err)
	return wstat.written, err
}

func prefixer(prefix string, w io.Writer) io.Writer {
	return &prefixWriter{
		w:          w,
		prefix:     []byte(prefix),
		needPrefix: true,
	}
}
