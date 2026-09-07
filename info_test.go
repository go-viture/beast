// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package beast

import (
	"errors"
	"testing"

	"github.com/go-macos/iokit/viture"
)

// TestATextReadIsAddressedToTheOtherHalfOfTheId.
//
// ⛔ THIS IS THE MISTAKE THAT COST THE FIRST ATTEMPT. Reusing command() sends
// DirRead -- 0x31 -- because command() decides the direction itself, and these
// ids live at 0x30xx. It also sends a length of 0x03, while the frame that
// actually answered carries NONE. Both had to be transcribed from the exchange
// that worked rather than adapted from the one beside it.
func TestATextReadIsAddressedToTheOtherHalfOfTheId(t *testing.T) {
	f := &fake{answers: []uint16{0}, texts: []string{"20.0.01.027_20260825"}}
	g := &Glasses{p: f}
	if _, err := g.text(MsgFirmwareVersion); err != nil {
		t.Fatalf("text: %v", err)
	}
	if len(f.dirs) != 1 || f.dirs[0] != DirText {
		t.Errorf("sent direction %#02x, want %#02x: 0x31 is a different family of ids",
			f.dirs[0], DirText)
	}
	sent := f.asked[0]
	want := []byte{viture.EventHeader, 0x00, MsgFirmwareVersion, DirText, 0x00, 0x00, 0x00, 0x00}
	for i, c := range want {
		if sent[i] != c {
			t.Fatalf("byte %d is %#02x, want %#02x\n got % x\nwant % x",
				i, sent[i], c, sent[:8], want)
		}
	}
	if len(sent) != viture.ReportSize {
		t.Errorf("the report is %d bytes, want %d", len(sent), viture.ReportSize)
	}
}

// What the headset answered, through the package.
func TestInfoIsWhatTheHeadsetSaid(t *testing.T) {
	// ⭐ MEASURED on a Beast, 2026-09-07. The package serial is EMPTY because
	// this headset answers 0xff repeated for it, which the frame model reads as
	// absent rather than as a serial made of 0xff.
	f := &fake{
		answers: []uint16{0, 0, 0},
		texts:   []string{"20.0.01.027_20260825", "R6PMCC613005G", ""},
	}
	in, err := (&Glasses{p: f}).Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if in.FirmwareVersion != "20.0.01.027_20260825" {
		t.Errorf("FirmwareVersion = %q", in.FirmwareVersion)
	}
	if in.BoardSerial != "R6PMCC613005G" {
		t.Errorf("BoardSerial = %q", in.BoardSerial)
	}
	if in.PackageSerial != "" {
		t.Errorf("PackageSerial = %q, want empty: this headset has none", in.PackageSerial)
	}
	// Asked in a fixed order, so a caller reading the transcript knows which
	// answer belongs to which question.
	wantIDs := []byte{MsgFirmwareVersion, MsgBoardSerial, MsgPackageSerial}
	for i, id := range wantIDs {
		if f.asked[i][2] != id {
			t.Errorf("question %d asked %#02x, want %#02x", i, f.asked[i][2], id)
		}
	}
}

// ⛔ NOTHING IS INVENTED WHEN THE HEADSET WILL NOT SAY. There is no second
// version to fall back on, and a wrong one on a screen is worse than a blank
// because nobody can tell it is wrong.
func TestInfoInventsNothing(t *testing.T) {
	boom := errors.New("the headset did not answer")

	t.Run("everything refused", func(t *testing.T) {
		in, err := (&Glasses{p: &fake{err: boom}}).Info()
		if !errors.Is(err, boom) {
			t.Errorf("Info = %v, want the transport error", err)
		}
		if in != (Info{}) {
			t.Errorf("Info = %+v, want everything empty", in)
		}
	})

	t.Run("no headset at all", func(t *testing.T) {
		var none *Glasses
		if _, err := none.text(MsgFirmwareVersion); !errors.Is(err, ErrNoDevice) {
			t.Errorf("text on a nil Glasses = %v, want ErrNoDevice", err)
		}
	})

	t.Run("one answer is enough to report success", func(t *testing.T) {
		// A partial answer is still an answer: the empty fields speak for
		// themselves, and failing the whole call would hide the one that came.
		f := &fake{answers: []uint16{0}, texts: []string{"20.0.01.027_20260825"}}
		in, err := (&Glasses{p: f}).Info()
		if err != nil {
			t.Errorf("Info = %v, want success when something was read", err)
		}
		if in.FirmwareVersion == "" {
			t.Error("the answer that arrived was dropped")
		}
	})
}
