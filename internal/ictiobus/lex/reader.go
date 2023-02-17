package lex

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"unicode/utf8"
)

// this is a reader that buffers as it goes so that we can 'undo' reads as
// needed. using regex lib on readers p much requires this unless the ONLY info
// required is "did it match", ugh.
//
// This reader implements io.ReadSeeker, io.RuneReader
type regexReader struct {
	b     []byte
	r     *bufio.Reader
	cur   int
	marks map[string]int
	atEOF bool
}

func NewRegexReader(r io.Reader) *regexReader {
	return &regexReader{
		b:     make([]byte, 0),
		r:     bufio.NewReader(r),
		marks: make(map[string]int),
	}
}

func (rr *regexReader) avail() int {
	return len(rr.b) - rr.cur
}

// reads from buffer and advances cursor by number of bytes read. if n bytes not
// avail, returns all bytes that ARE avail.
func (rr *regexReader) readBuf(n int) []byte {
	limit := rr.avail()
	if n < limit {
		limit = n
	}

	read := rr.b[rr.cur : rr.cur+limit]
	rr.cur += limit
	return read
}

// calls Read on underlying reader to attempt to read n bytes into the buffer.
// buffers all bytes read and returns the error. does not modify the cursor.
func (rr *regexReader) readIntoBuf(n int) (actualRead int, err error) {
	read := make([]byte, n)

	actualRead, err = rr.r.Read(read)
	// if we read at least 1 byte for ANY reason even if we also got an error,
	// we must buffer it
	if actualRead > 0 {
		rr.b = append(rr.b, read[:actualRead]...)
	}

	return actualRead, err
}

// NextRune reads and discards the next n runes.
// returns the number of bytes read in total and the first error encountered, if
// any.
func (rr *regexReader) NextRune(count int) (size int, err error) {
	totalRead := 0
	for i := 0; i < count; i++ {
		_, size, err := rr.ReadRune()
		totalRead += size
		if err != nil {
			// either it's EOF or something we cannot handle; either way,
			// immediately return
			return totalRead, err
		}
	}
	return totalRead, nil
}

// SearchAndAdvance applies the given regular expression and moves the internal
// cursor forward exactly 1 byte ahead of the location of the searched-for term.
// If no term is found, the cursor is not advanced at all and an empty/nil slice
// will be returned; otherwise, the return value is a slice of matches where
// the index of each match is the contents of that sub-expression group, and
// group 0 is the entire match.
//
// uses (and will overwrite) mark called "SEARCH_AND_ADVANCE"
//
// returns io.EOF as error value if at the end of the stream. []string will
// always be nil if at EOF; that is, the reader can never detect that it is at
// EOF until there is a failure to match, so any successful match will result in
// a nil-error and non-nil matches.
func (rr *regexReader) SearchAndAdvance(re *regexp.Regexp) ([]string, error) {
	// if we KNOW we are at the end, no reason to attempt a match. immediately
	// return io.EOF.

	rr.Mark("SEARCH_AND_ADVANCE")
	matchIndexes := re.FindReaderSubmatchIndex(rr)
	matches := rr.GetMatches("SEARCH_AND_ADVANCE", matchIndexes)
	rr.Restore("SEARCH_AND_ADVANCE")
	if len(matches) > 0 {
		rr.Seek(int64(matchIndexes[1]), io.SeekCurrent)
	} else {
		// is it because we got an error while reading the underlying reader?
		// if so, we need to stop reading

		// go to end of buffer:
		_, err := rr.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, fmt.Errorf("seeking to end of buffer: %w", err)
		}

		// try to read one more byte
		_, err = rr.Read(make([]byte, 1))

		// if we got an eof there, return it and remember it for fast returning
		// next time
		if err == io.EOF {
			rr.atEOF = true
		}

		if err != nil {
			return nil, err
		}

		// no error? great. it's a plain no-match. go back to our mark
		rr.Restore("SEARCH_AND_ADVANCE")
	}
	return matches, nil
}

// GetMatches attempts to read the strings located in the buffered
// contents from inclusive byte index from to non-inclusive byte index to,
// relative to the provided mark. This is mostly designed to be able to retrieve
// the results of matches detected by regexp.FindReaderSubmatchIndex.
//
// To use, call Mark() with some name. Immediately after, call
// regexp.FindReaderSubmatchIndex on this reader. Then, pass the returned pairs
// and the desired group to retrieve to this function, along with the name of
// the mark originally set.
//
// returns a slice where every entry is a string. Its position in the slice
// corresponds to the group number it is in. If a sub-expression did not match,
// the string will be empty. If there was no match at all, the returned slice
// will be nil. Group 0 is the entire match.
func (rr *regexReader) GetMatches(mark string, pairs []int) []string {
	markOffset, ok := rr.marks[mark]
	if !ok {
		panic(fmt.Sprintf("invalid mark name: %q", mark))
	}

	if pairs == nil || len(pairs) == 0 {
		return nil
	}

	matches := make([]string, len(pairs)/2)
	matches[0] = string(rr.b[markOffset+pairs[0] : markOffset+pairs[1]])

	for i := 2; i < len(pairs); i += 2 {
		left := pairs[i]
		right := pairs[i+1]
		if left != -1 && right != -1 {
			matches[i/2] = string(rr.b[markOffset+left : markOffset+right])
		}
	}

	return matches
}

func (rr *regexReader) ReadRune() (r rune, size int, err error) {
	// okay, so, read 1 single byte. assuming it is a utf-8 byte, we can
	// instantly tell how many more bytes are needed by reading the first few
	// bits of the byte.
	charBytes := make([]byte, 1)
	n, err := rr.Read(charBytes)
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
		n, err := rr.Read(additionalCharBytes)
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
		rr.cur -= missedBy
	}

	return r, size, setErr
}

// Mark creates a new marker with the given name, for later use with Restore, at
// the current offset.
func (rr *regexReader) Mark(name string) {
	rr.marks[name] = rr.cur
}

// Restore seeks back to the marker with the given name. Panics if the name
// doesn't exist.
func (rr *regexReader) Restore(name string) {
	offset, ok := rr.marks[name]
	if !ok {
		panic(fmt.Sprintf("invalid mark name: %q", name))
	}

	rr.cur = offset
}

// Offset returns the current absolute offset into the buffered bytes that the
// reader is currently at. The returned number, if passed into Seek with a
// whence of SeekStart, would make the reader go back to this exact position.
func (rr *regexReader) Offset() int64 {
	return int64(rr.cur)
}

func (rr *regexReader) Read(p []byte) (n int, err error) {
	// do we already have |p| bytes at cursor location?
	read := rr.readBuf(len(p))
	stillNeed := len(p) - len(read)

	if stillNeed > 0 {
		// need to make this much avail.
		var actualRead int
		actualRead, err = rr.readIntoBuf(stillNeed)
		if actualRead > 0 {
			readAdd := rr.readBuf(actualRead)
			read = append(read, readAdd...)
		}
	}

	// we've now read everyfin we can. copy it to p.
	n = len(read)
	copy(p, read)
	return n, err
}

// Seek moves the internal cursor to the provided offset. As seekableReader
// itself reads from an underlying Reader whose end is unknown, SeekEnd will be
// interpreted as relative to the end of the *buffered* bytes, not those in the
// underlying reader.
func (rr *regexReader) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	if whence == io.SeekStart {
		newOffset = offset
	} else if whence == io.SeekCurrent {
		newOffset = int64(rr.cur) + offset
	} else if whence == io.SeekEnd {
		newOffset = int64(len(rr.b)) + offset
	} else {
		return 0, fmt.Errorf("unknown whence argument: %v", whence)
	}

	if newOffset < 0 {
		return 0, fmt.Errorf("resulting absolute offset specifies index before start of file: %d", newOffset)
	}
	if newOffset > int64(len(rr.b)) {
		newOffset = int64(len(rr.b))
	}

	rr.cur = int(newOffset)
	return newOffset, nil
}
