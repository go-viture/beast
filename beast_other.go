// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build !darwin

package beast

import "github.com/go-macos/iokit/viture"

// platform is the seam the portable model is tested against.
type platform interface {
	exchange(msg, dir byte, report []byte) (viture.Event, error)
	close() error
}

// open reports [ErrUnsupported].
//
// The frames and the exchange logic above are portable and tested everywhere;
// only the HID transport is macOS for now. Saying so beats handing back a
// headset that answers nothing, which reads like a headset nobody plugged in.
func open() (*Glasses, error) { return nil, ErrUnsupported }
