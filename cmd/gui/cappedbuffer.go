package main

import (
	"sync"
	"unicode/utf8"
)

// maxRcloneStderrBytes bounds how much of one rclone process's stderr is
// kept in memory. A mount can stay alive for days, and rclone keeps
// writing errors/warnings the whole time; an unbounded buffer (the old
// bytes.Buffer) only ever grew until the process exited.
const maxRcloneStderrBytes = 64 * 1024

// cappedOmittedNotice is prefixed to String()'s result once any older
// output has been discarded, so the reader knows the beginning is missing.
const cappedOmittedNotice = "…(앞부분 생략)\n"

// cappedBuffer is an io.Writer that only ever keeps the most recent max
// bytes written to it. rclone's *last* output is what explains a failure
// (the fatal error is printed right before it exits), so keeping the tail
// loses nothing that the failure dialog needs.
type cappedBuffer struct {
	mu      sync.Mutex
	max     int
	buf     []byte
	dropped bool // true once any older output has been discarded
}

func newCappedBuffer(max int) *cappedBuffer {
	if max <= 0 {
		max = 1
	}
	return &cappedBuffer{max: max}
}

// Write always reports the full length as written and never fails —
// exec.Cmd treats a short write on Stderr as an error that would break
// the process's output copying.
func (b *cappedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	// Compact lazily (only past 2x the cap) so a burst of small writes
	// doesn't copy the whole buffer every time.
	if len(b.buf) > 2*b.max {
		keep := make([]byte, b.max)
		copy(keep, b.buf[len(b.buf)-b.max:])
		b.buf = keep
		b.dropped = true
	}
	return len(p), nil
}

// String returns at most max bytes of the most recent output, never
// starting in the middle of a multi-byte character.
func (b *cappedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	data := b.buf
	truncated := b.dropped
	if len(data) > b.max {
		data = data[len(data)-b.max:]
		truncated = true
	}
	if truncated {
		for len(data) > 0 && !utf8.RuneStart(data[0]) {
			data = data[1:]
		}
		return cappedOmittedNotice + string(data)
	}
	return string(data)
}
