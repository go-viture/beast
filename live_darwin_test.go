// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build darwin

package beast

import (
	"errors"
	"os"
	"testing"

	"github.com/go-macos/iokit/viture"
)

// TestLiveRead asks a real Beast what it is doing.
//
// ⛔ READS ONLY. Set BEAST_LIVE=1 with a Beast attached. Nothing here writes:
// a read changes nothing, repeats safely, and its success is visible -- which
// is why it is the first thing to try on any device, and why nineteen writes
// judged by a screen taught that lesson the expensive way.
func TestLiveRead(t *testing.T) {
	if os.Getenv("BEAST_LIVE") == "" {
		t.Skip("set BEAST_LIVE=1 with a Beast attached")
	}
	g, err := Open()
	if err != nil {
		if errors.Is(err, ErrNoDevice) {
			t.Skip("no Beast attached")
		}
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = g.Close() }()

	// ⭐ THE NATIVE MODE IS THE ONE THAT ANSWERS IN THE DOCUMENTED VOCABULARY.
	// A read of MsgDisplayMode returns something in an encoding nobody has
	// explained; this one lands on the published constants and was confirmed
	// against the panel.
	native, err := g.Get(viture.MsgNativeDisplayMode)
	if err != nil {
		t.Fatalf("native display mode: %v", err)
	}
	t.Logf("native display mode: %#02x", native)

	// And a setting that is a state rather than a fact: it has been seen to
	// vanish and come back on the same cable.
	if v, err := g.Get(0x22); err != nil {
		t.Logf("brightness: %v (it has gone missing before and come back)", err)
	} else {
		t.Logf("brightness: %d", v)
	}

	// An id nothing lives at must say so rather than hang.
	if _, err := g.Get(0x55); !errors.Is(err, ErrNoSetting) {
		t.Errorf("an empty id answered %v, want ErrNoSetting", err)
	}
}
