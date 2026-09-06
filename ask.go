// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package beast

import (
	"errors"
	"fmt"

	"github.com/go-macos/iokit/viture"
)

// Status is what the headset answers to a write.
//
// ⭐ IT ANSWERS EVERY ONE, and that is the whole reason this package judges by
// the reply rather than by the display. Nineteen writes were lost watching a
// screen: a screen that does not change cannot tell a refusal from a command
// that never arrived, and one read settled in a second what those nineteen had
// not.
type Status uint16

// What the headset says back.
const (
	// StatusOK means the command was taken.
	StatusOK Status = 0
	// StatusRefused means the headset understood and said no -- which is how
	// its own limits were mapped: brightness accepted 0 to 8 and refused 9,
	// with nothing to watch and nobody guessing.
	StatusRefused Status = 4
	// StatusTooShort means the command carried fewer than two payload bytes.
	StatusTooShort Status = 6
)

// String renders a status the way an error message should read.
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "taken"
	case StatusRefused:
		return "refused"
	case StatusTooShort:
		return "too short"
	default:
		return fmt.Sprintf("code %d", uint16(s))
	}
}

// Err turns a status into an error, or nil.
func (s Status) Err() error {
	if s == StatusOK {
		return nil
	}
	return fmt.Errorf("beast: the headset %s", s)
}

// Get reads one setting and returns its value.
//
// ⛔ A READ CHANGES NOTHING, which is what makes it the right first experiment
// on any device: it is safe to repeat, and its success is visible in a way a
// command's is not.
//
// ⭐ AND A READ ANSWERING 2 MEANS "NO SUCH MESSAGE", which makes the whole id
// space discoverable without ever writing. Measured: twenty messages exist
// between 0x00 and 0x7f and every other id answers 2.
func (g *Glasses) Get(msg byte) (uint16, error) {
	if g == nil {
		return 0, ErrNoDevice
	}
	e, err := g.p.exchange(msg, viture.DirRead, command(msg, 0x03, 0))
	if err != nil {
		return 0, err
	}
	if e.Value == 2 {
		return 0, fmt.Errorf("%w: %#02x", ErrNoSetting, msg)
	}
	return e.Value, nil
}

// ErrNoSetting means the headset answered that it has no such message.
//
// ⚠ AND IT IS A STATE, NOT A FACT ABOUT THE PROTOCOL. Measured 2026-09-05:
// 0x22 and 0x30 both answered a read at 17:00 and both were gone by 18:46,
// with the same cable and no replug -- while every other id still answered.
// 0x22 came back the next morning. So report it; do not compile it in.
var ErrNoSetting = errors.New("beast: the headset has no such setting")

// Set writes one setting and reports what the headset said about it.
//
// ⛔ THE VALUE GOES IN TWICE, LITTLE-ENDIAN. That one detail is what nineteen
// attempts had wrong: the headset's own REPLIES carry a value little-endian and
// then big-endian, and copying that shape into a command has it refused. A
// reply and a command are not the same frame.
func (g *Glasses) Set(msg byte, value uint16) error {
	if g == nil {
		return ErrNoDevice
	}
	e, err := g.p.exchange(msg, viture.DirWrite, command(msg, 0x02, value))
	if err != nil {
		return err
	}
	return Status(e.Value).Err()
}

// Nudge moves a setting by one step, clamped, and says where it ended up.
//
// ⛔ IT READS FIRST. A key that means "brighter" has to know what it is
// brighter than, and the headset is the only thing that knows: somebody may
// have used the buttons on the arm since anything here last looked.
func (g *Glasses) Nudge(msg byte, by int, max uint16) (uint16, error) {
	at, err := g.Get(msg)
	if err != nil {
		return 0, err
	}
	next := int(at) + by
	if next < 0 {
		next = 0
	}
	if next > int(max) {
		next = int(max)
	}
	if uint16(next) == at {
		return at, nil
	}
	if err := g.Set(msg, uint16(next)); err != nil {
		return at, err
	}
	return uint16(next), nil
}

// command builds a report: the frame model lives in go-macos/iokit/viture,
// which decoded it, and is not copied here.
func command(msg, length byte, value uint16) []byte {
	b := make([]byte, viture.ReportSize)
	dir := viture.DirWrite
	if length == 0x03 && value == 0 {
		dir = viture.DirRead
	}
	copy(b, []byte{
		viture.EventHeader, 0x00, msg, dir, length, 0x00,
		byte(value), byte(value >> 8),
		byte(value), byte(value >> 8),
	})
	return b
}
