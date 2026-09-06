// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build darwin

package beast

import (
	"context"
	"fmt"
	"time"

	"github.com/go-macos/iokit/hid"
	"github.com/go-macos/iokit/viture"
)

// platform is the seam the portable model is tested against.
type platform interface {
	exchange(msg, dir byte, report []byte) (viture.Event, error)
	close() error
}

// device is the real one.
type device struct{ d *hid.Device }

func open() (*Glasses, error) {
	devs, err := hid.Devices(hid.Filter{VendorID: VendorID, UsagePage: UsagePage})
	if err != nil {
		return nil, fmt.Errorf("beast: looking for the headset: %w", err)
	}
	if len(devs) == 0 {
		return nil, ErrNoDevice
	}
	d := devs[0]
	if err := d.Open(); err != nil {
		return nil, fmt.Errorf("beast: opening the headset: %w", err)
	}
	return &Glasses{p: &device{d: d}}, nil
}

// exchange listens, asks, and returns the frame that answers.
//
// ⛔ IT LISTENS BEFORE IT ASKS. The headset answers in milliseconds; a listener
// started afterwards is a listener that misses its own answer. This was the
// shape of one of the earliest mistakes here.
//
// ⛔ AND IT MATCHES ON BOTH THE MESSAGE AND THE DIRECTION. A reply carries the
// request's direction with the reply bit set -- a read 0x31 comes back 0x51, a
// write 0x01 comes back 0x21 -- so an unrelated announcement about the same
// setting, which this device does send, is not mistaken for an answer.
func (x *device) exchange(msg, dir byte, report []byte) (viture.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Wait)
	in := make(chan viture.Event, 16)
	streamed := make(chan struct{})
	go func() {
		defer close(streamed)
		_ = hid.Stream(ctx, func(_ *hid.Device, b []byte) {
			if e, ok := viture.ParseEvent(b); ok {
				select {
				case in <- e:
				default:
				}
			}
		}, x.d)
	}()
	// ⛔ CANCEL, WAIT, then let go. Closing or returning while the stream still
	// holds the device is what ends the process rather than the call.
	defer func() {
		cancel()
		<-streamed
	}()
	time.Sleep(80 * time.Millisecond)

	if err := x.d.SetReport(hid.Output, 0, report); err != nil {
		return viture.Event{}, fmt.Errorf("beast: asking the headset: %w", err)
	}
	want := dir + viture.ReplyBit
	deadline := time.After(Wait)
	for {
		select {
		case e := <-in:
			if e.ID == msg && e.Kind == want {
				return e, nil
			}
		case <-deadline:
			return viture.Event{}, fmt.Errorf("%w about %#02x", ErrNoAnswer, msg)
		}
	}
}

func (x *device) close() error { return x.d.Close() }
