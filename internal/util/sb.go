package util

import "strings"

type stringBuilderOp struct {
	writeArg       []byte
	isWrite        bool    // if true the op is Write (can't just use non-nil writeArg because nil is a valid argument)
	isReset        bool    // if true the op is Reset
	writeByteArg   *byte   // if not nil the op is WriteByte
	writeRuneArg   *rune   // if not nil the op is WriteRune
	writeStringArg *string // if not nil the op is WriteString
}

// UndoableStringBuilder wraps strings.Builder and provides an additional
// 'Undo' function which undoes the prior mutation.
//
// Note that to accomplish this it more or less saves a complete copy of every
// operation and lazily applies them only when needed. Further, in order to
// preserve the undo operation after calling a function other than one of the
// Write functions, it's possible that it will need to lazily apply multiple
// times. Take care when calling functions other than the Write functions.
type UndoableStringBuilder struct {
	cache      *strings.Builder // set to nil to invalidate cache
	pendingOps []stringBuilderOp
	cur        int // undo cursor
}

// Len returns the number of accumulated bytes; b.Len() == len(b.String()).
//
// This will force evaluation of all pending operations if it hasn't been
// applied yet or if there have been mutation functions called on the
// UndoableStringBuilder since the last time pending operations were applied.
func (usb *UndoableStringBuilder) Len() int {
	usb.apply()
	return usb.cache.Len()
}

// Reset resets the UndoableStringBuilder to be empty.
//
// This is an undoable operation.
func (usb *UndoableStringBuilder) Reset() {
	op := stringBuilderOp{
		isReset: true,
	}

	usb.addOp(op)
}

// String returns the accumulated string.
//
// This will force evaluation of all pending operations if it hasn't been
// applied yet or if there have been mutation functions called on the
// UndoableStringBuilder since the last time pending operations were applied.
func (usb *UndoableStringBuilder) String() string {
	usb.apply()
	return usb.cache.String()
}

// Write appends the contents of p to b's buffer. Write always returns len(p),
// nil, but the return values are kept to implement the io.Writer interface.
//
// This is an undoable operation.
func (usb *UndoableStringBuilder) Write(p []byte) (int, error) {
	op := stringBuilderOp{
		isWrite:  true,
		writeArg: p,
	}

	usb.addOp(op)

	return len(p), nil
}

// WriteByte appends the byte c to b's buffer. The returned error is always nil
// but is kept to implement the io.ByteWriter interface.
//
// This is an undoable operation.
func (usb *UndoableStringBuilder) WriteByte(c byte) error {
	op := stringBuilderOp{
		writeByteArg: &c,
	}

	usb.addOp(op)

	return nil
}

// WriteRune appends the UTF-8 encoding of Unicode code point r to b's buffer.
//
// This is an undoable operation.
func (usb *UndoableStringBuilder) WriteRune(r rune) {
	op := stringBuilderOp{
		writeRuneArg: &r,
	}

	usb.addOp(op)
}

// WriteString appends the contents of s to b's buffer.
//
// This is an undoable operation.
func (usb *UndoableStringBuilder) WriteString(s string) {
	op := stringBuilderOp{
		writeStringArg: &s,
	}

	usb.addOp(op)
}

// Undo reverts the previous operation. This undo can be later reverted with
// Redo() provided that no further mutation operations have been called.
//
// Undo can be called multiple times to undo as many operations as are desired.
// If called when there are no further operations to undo, this function has no
// effect.
func (usb *UndoableStringBuilder) Undo() {
	usb.cur--
	if usb.cur < 0 {
		usb.cur = 0
	}
}

// Redo reverts the previous Undo. Redo can be called multiple times to revert
// as many Undos as are desired. If called when there are no further undos to
// revert, this function has no effect.
func (usb *UndoableStringBuilder) Redo() {
	usb.cur++
	if usb.cur > len(usb.pendingOps) {
		usb.cur = len(usb.pendingOps)
	}
}

func (usb *UndoableStringBuilder) addOp(op stringBuilderOp) {
	usb.cache = nil // invalidate cache

	// set pending ops to remove any ops we have undone, by adding an op they
	// are no longer re-doable.
	if usb.pendingOps != nil && usb.cur < len(usb.pendingOps) {
		usb.pendingOps = usb.pendingOps[:usb.cur]
	}

	usb.pendingOps = append(usb.pendingOps, op)

	// advance the cursor so it points to the current end after the append
	usb.cur++
}

func (usb *UndoableStringBuilder) apply() {
	// dont apply operations if the cache is not invalid
	if usb.cache != nil {
		return
	}

	sb := strings.Builder{}

	for i := range usb.pendingOps {
		// only go up to the current undo pointer
		if i >= usb.cur {
			break
		}

		op := usb.pendingOps[i]

		if op.isWrite {
			sb.Write(op.writeArg)
		} else if op.writeByteArg != nil {
			sb.WriteByte(*op.writeByteArg)
		} else if op.writeRuneArg != nil {
			sb.WriteRune(*op.writeRuneArg)
		} else if op.writeStringArg != nil {
			sb.WriteString(*op.writeStringArg)
		} else if op.isReset {
			sb.Reset()
		} else {
			panic("invalid sbOp; should never happen")
		}
	}

	usb.cache = &sb
}
