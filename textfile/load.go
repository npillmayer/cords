package textfile

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"unicode/utf8"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/cords/chunk"
)

// Some constants for fragment size defaults.
const (
	twoKb     = 2048
	sixKb     = 6144
	tenKb     = 10240
	hundredKb = 1024000
	oneMb     = 1048576
)

// textFile represents an OS file to be loaded as a cord.
type textFile struct {
	path string
	info os.FileInfo
	file *os.File
}

// Load reads a UTF-8 text file and materializes it as a cord.
//
// This migration step intentionally uses a synchronous, correctness-first load
// path on top of the chunk/sum-tree cord implementation.
//
// `initialPos` and `wg` are currently accepted for API compatibility and are
// ignored in this synchronous implementation.
//
// `fragSize` controls the read buffer size used during loading. If it is out of
// range, a default based on file size is chosen.
func Load(name string, initialPos, fragSize int64, wg *sync.WaitGroup) (cords.Cord, error) {
	_ = initialPos
	if wg != nil {
		wg.Add(1)
		defer wg.Done()
	}

	tf, err := openFile(name)
	if err != nil {
		return cords.Cord{}, err
	}
	defer func() {
		_ = tf.file.Close()
	}()

	tracer().Infof("opened file %s", tf.info.Name())
	fragSize = normalizeFragSize(fragSize, tf.info.Size())

	if tf.info.Size() == 0 {
		return cords.Cord{}, nil
	}

	b := cords.NewBuilder()
	if err := loadWithPrefetch(tf.file, fragSize, b); err != nil {
		return cords.Cord{}, err
	}
	return b.Cord(), nil
}

// openFile opens an OS file and checks basic preconditions.
func openFile(name string) (*textFile, error) {
	fi, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("file is not a regular file")
	}
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return &textFile{
		path: name,
		info: fi,
		file: file,
	}, nil
}

func normalizeFragSize(fragSize, fileSize int64) int64 {
	if fragSize > 0 && fragSize <= tenKb {
		return fragSize
	}
	switch {
	case fileSize <= 0:
		return twoKb
	case fileSize < 64:
		return fileSize
	case fileSize < 1024:
		return 64
	case fileSize < tenKb:
		return 1024
	case fileSize < hundredKb:
		return 512
	case fileSize < oneMb:
		return twoKb
	default:
		return sixKb
	}
}

func loadWithPrefetch(file *os.File, fragSize int64, b *cords.Builder) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chunks := make(chan []byte, 8)
	errCh := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer close(chunks)
		readFileChunks(ctx, file, fragSize, chunks, errCh)
	}()

	for frag := range chunks {
		if err := b.AppendBytes(frag); err != nil {
			cancel()
			<-done
			if readErr := consumeErr(errCh); readErr != nil {
				return readErr
			}
			return err
		}
	}
	<-done
	if err := consumeErr(errCh); err != nil {
		return err
	}
	return nil
}

func readFileChunks(ctx context.Context, file *os.File, fragSize int64, out chan<- []byte, errCh chan<- error) {
	reader := io.Reader(file)
	buf := make([]byte, fragSize)
	pending := make([]byte, 0, 3)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			data := append(pending, buf[:n]...)
			prefix, tail, splitErr := splitValidUTF8Prefix(data)
			if splitErr != nil {
				publishErr(errCh, splitErr)
				return
			}
			if len(prefix) > 0 {
				frag := append([]byte(nil), prefix...)
				select {
				case out <- frag:
				case <-ctx.Done():
					return
				}
			}
			pending = pending[:0]
			pending = append(pending, tail...)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			publishErr(errCh, fmt.Errorf("textfile load failed: %w", readErr))
			return
		}
	}
	if len(pending) > 0 {
		if !utf8.Valid(pending) {
			publishErr(errCh, chunk.ErrInvalidUTF8)
			return
		}
		frag := append([]byte(nil), pending...)
		select {
		case out <- frag:
		case <-ctx.Done():
			return
		}
	}
}

func publishErr(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}

func consumeErr(errCh <-chan error) error {
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func splitValidUTF8Prefix(data []byte) (prefix []byte, tail []byte, err error) {
	if len(data) == 0 {
		return nil, nil, nil
	}
	if utf8.Valid(data) {
		return data, nil, nil
	}
	maxTail := 3
	if len(data) < maxTail {
		maxTail = len(data)
	}
	for tailLen := 1; tailLen <= maxTail; tailLen++ {
		cut := len(data) - tailLen
		if utf8.Valid(data[:cut]) && !utf8.FullRune(data[cut:]) {
			return data[:cut], data[cut:], nil
		}
	}
	return nil, nil, chunk.ErrInvalidUTF8
}
