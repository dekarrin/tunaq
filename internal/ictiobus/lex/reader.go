package lex

import (
	"bufio"
	"fmt"
	"io"
	"unicode/utf8"
)

// this is a reader that buffers as it goes so that we can 'undo' reads as
// needed. using regex lib on readers p much requires this unless the ONLY info
// required is "did it match", ugh.
//
// This reader implements io.ReadSeeker, io.RuneReader
type seekableReader struct {
	b     []byte
	r     *bufio.Reader
	cur   int
	marks map[string]int
}

func NewSeekableReader(r io.Reader) *seekableReader {
	return &seekableReader{
		b:     make([]byte, 0),
		r:     bufio.NewReader(r),
		marks: make(map[string]int),
	}
}

func (sr *seekableReader) avail() int {
	return len(sr.b) - sr.cur
}

// reads from buffer and advances cursor by number of bytes read. if n bytes not
// avail, returns all bytes that ARE avail.
func (sr *seekableReader) readBuf(n int) []byte {
	limit := sr.avail()
	if n < limit {
		limit = n
	}

	read := sr.b[sr.cur : sr.cur+limit]
	sr.cur += limit
	return read
}

// calls Read on underlying reader to attempt to read n bytes into the buffer.
// buffers all bytes read and returns the error. does not modify the cursor.
func (sr *seekableReader) readIntoBuf(n int) (actualRead int, err error) {
	read := make([]byte, n)

	actualRead, err = sr.r.Read(read)
	// if we read at least 1 byte for ANY reason even if we also got an error,
	// we must buffer it
	if actualRead > 0 {
		sr.b = append(sr.b, read[:actualRead]...)
	}

	return actualRead, err
}

// GetBufferedString attempts to read the string located in the buffered
// contents from inclusive byte index from to non-inclusive byte index to. This
// is mostly designed to be able to retrieve the results of a string detected by
// regexp.FindReaderSubmatchIndex.
//
// from and to must exist in the buffer. If this is called with a pair from the
// results of FindReaderSubmatchIndex called on this reader, they are gauranteed
// to be available.
func (sr *seekableReader) GetBufferedString(from int, to int) (string, error) {
	if to > len(sr.b) {
		return "", fmt.Errorf("to index is past end of buffered bytes: %d", to)
	}
	if from > len(sr.b) {
		return "", fmt.Errorf("from index is past end of buffered bytes: %d", from)
	}
	if to < 0 {
		return "", fmt.Errorf("to < 0: %d", to)
	}
	if from < 0 {
		return "", fmt.Errorf("from < 0: %d", from)
	}

	str := string(sr.b[from:to])
	return str, nil
}

func (sr *seekableReader) ReadRune() (r rune, size int, err error) {
	// okay, so, read 1 single byte. assuming it is a utf-8 byte, we can
	// instantly tell how many more bytes are needed by reading the first few
	// bits of the byte.
	charBytes := make([]byte, 1)
	n, err := sr.Read(charBytes)
	if n != 1 {
		return r, size, err
	}

	var setErr error
	if err != nil {
		setErr = err
	}

	firstByte := charBytes[0]
	var remBytes int

	if firstByte>>7 == 0 {
		// 0xxxxxxx, 1-byte rune
		remBytes = 0
	} else if firstByte>>5 == 0b110 {
		// 110xxxxx, 2-byte rune
		remBytes = 1
	} else if firstByte>>4 == 0b1110 {
		// 1110xxxx, 3-byte rune
		remBytes = 2
	} else if firstByte>>3 == 0b11110 {
		// 11110xxx, 4-byte rune
		remBytes = 3
	}

	if remBytes > 0 {
		if setErr != nil && setErr != io.EOF {
			// we had a non-eof error, we cannot read further. stop.
			return r, n, setErr
		}
		additionalCharBytes := make([]byte, remBytes)
		n, err := sr.Read(additionalCharBytes)
		if n != remBytes {
			if err == io.EOF {
				return r, n, fmt.Errorf("couldn't read all bytes of utf-8 character")
			}
			return r, n, err
		}
		setErr = err
		charBytes = append(charBytes, additionalCharBytes...)
	}

	// we now (should) have a full rune ready. decode it.
	r, size = utf8.DecodeRune(charBytes)

	// if size is not the exact number of runes read, we've made a mistake.
	// enshore that the cursor backs up to fix this
	missedBy := len(charBytes) - size
	if missedBy > 0 {
		sr.cur -= missedBy
	}

	return r, size, setErr
}

// Mark creates a new marker with the given name, for later use with Restore, at
// the current offset.
func (sr *seekableReader) Mark(name string) {
	sr.marks[name] = sr.cur
}

// Restore seeks back to the marker with the given name. Panics if the name
// doesn't exist.
func (sr *seekableReader) Restore(name string) {
	offset, ok := sr.marks[name]
	if !ok {
		panic(fmt.Sprintf("invalid mark name: %q", name))
	}

	sr.cur = offset
}

// Offset returns the current absolute offset into the buffered bytes that the
// reader is currently at. The returned number, if passed into Seek with a
// whence of SeekStart, would make the reader go back to this exact position.
func (sr *seekableReader) Offset() int64 {
	return int64(sr.cur)
}

func (sr *seekableReader) Read(p []byte) (n int, err error) {
	// do we already have |p| bytes at cursor location?
	read := sr.readBuf(len(p))
	stillNeed := len(p) - len(read)

	if stillNeed > 0 {
		// need to make this much avail.
		var actualRead int
		actualRead, err = sr.readIntoBuf(stillNeed)
		if actualRead > 0 {
			readAdd := sr.readBuf(actualRead)
			read = append(read, readAdd...)
		}
	}

	// we've now read everyfin we can. copy it to p.
	n = len(read)
	copy(p, read)
	return n, err
}

// Mark

// Seek moves the internal cursor to the provided offset. As seekableReader
// itself reads from an underlying Reader whose end is unknown, SeekEnd will be
// interpreted as relative to the end of the *buffered* bytes, not those in the
// underlying reader.
func (sr *seekableReader) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	if whence == io.SeekStart {
		newOffset = offset
	} else if whence == io.SeekCurrent {
		newOffset = int64(sr.cur) + offset
	} else if whence == io.SeekEnd {
		newOffset = int64(len(sr.b)) + offset
	} else {
		return 0, fmt.Errorf("unknown whence argument: %v", whence)
	}

	if newOffset < 0 {
		return 0, fmt.Errorf("resulting absolute offset specifies index before start of file: %d", newOffset)
	}
	if newOffset > int64(len(sr.b)) {
		newOffset = int64(len(sr.b))
	}

	sr.cur = int(newOffset)
	return newOffset, nil
}
