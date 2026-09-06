// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

// Package beast talks to a VITURE Beast headset in pure Go, with
// CGO_ENABLED=0.
//
// ⛔ IT IS NOT THE SAME PROTOCOL AS THE LUMA, and the two do not even share an
// envelope: a Beast frame begins 0x10 and a Luma frame begins FA 55. Habits do
// not carry across. What they do share is a house style -- ask before you
// write, and judge by what the device answers rather than by what the screen
// does.
//
// ⭐ THE HEADSET ANSWERS EVERY COMMAND WITH A STATUS, and that is worth more
// than watching the display: a screen that does not change cannot tell a
// refusal from a command that never arrived. Finding that out took nineteen
// failed writes.
//
// The frames themselves live in go-macos/iokit/viture, which decoded them.
// This package is the transport and the ergonomics on top: open the right
// interface, listen before asking, and hand back what the headset said.
package beast

import (
	"errors"
	"time"
)

// ErrUnsupported is returned off macOS.
var ErrUnsupported = errors.New("beast: only macOS is wired up")

// ErrNoDevice says no Beast control interface is on the bus.
var ErrNoDevice = errors.New("beast: no VITURE Beast found")

// ErrNoAnswer says the headset did not reply in time.
var ErrNoAnswer = errors.New("beast: the headset did not answer")

// The device this speaks to.
//
// ⭐ 35ca:1201 ON THE VENDOR USAGE PAGE is the control interface, and it is the
// only one of the three the glasses publish that carries the protocol: the
// others are called "VITURE Microphone" and are a Consumer-page audio set. The
// name misleads and the usage page does not -- a mistake this fleet has now
// made in both directions.
const (
	VendorID  uint16 = 0x35ca
	UsagePage uint16 = 0xff00
)

// Wait is how long the headset is given to answer.
//
// Generous for a device that answers in milliseconds, and short enough that a
// menu row does not appear to hang.
const Wait = 2 * time.Second

// Glasses is one open headset.
type Glasses struct {
	p platform
}

// Open finds the Beast's control interface and claims it.
func Open() (*Glasses, error) { return openDevice() }

// openDevice is the platform call, replaced in tests so that everything above
// the transport is exercised on every operating system.
var openDevice = open

// Close releases it.
//
// ⛔ IT WAITS FOR THE LISTENER. Closing a HID device while a stream still holds
// it does not fail, it ends the PROCESS -- macOS kills the program with
// "os_unfair_lock is corrupt", no Go panic, no stack, nothing on the program's
// own output. go-macos/iokit refuses that now; this does it in the right order
// anyway.
func (g *Glasses) Close() error {
	if g == nil {
		return nil
	}
	return g.p.close()
}
