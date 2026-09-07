// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package beast

import (
	"errors"
	"testing"

	"github.com/go-macos/iokit/viture"
)

// fake stands in for the headset: it records what was asked and answers from a
// script.
type fake struct {
	asked   [][]byte
	dirs    []byte
	answers []uint16
	// texts answers the text ids, in order, when set.
	texts []string
	err   error
}

func (f *fake) exchange(msg, dir byte, report []byte) (viture.Event, error) {
	f.asked = append(f.asked, append([]byte(nil), report...))
	f.dirs = append(f.dirs, dir)
	if f.err != nil {
		return viture.Event{}, f.err
	}
	if len(f.answers) == 0 {
		return viture.Event{}, ErrNoAnswer
	}
	v := f.answers[0]
	f.answers = f.answers[1:]
	text := ""
	if len(f.texts) > 0 {
		text, f.texts = f.texts[0], f.texts[1:]
	}
	return viture.Event{ID: msg, Kind: dir + viture.ReplyBit, Value: v, Text: text}, nil
}
func (f *fake) close() error { return nil }

// TestAWriteCarriesTheValueTwiceLittleEndian.
//
// ⛔ THIS IS THE DETAIL NINETEEN ATTEMPTS HAD WRONG. The headset's own REPLIES
// carry a value little-endian and then big-endian, and copying that shape into
// a command has it refused. A reply and a command are not the same frame.
func TestAWriteCarriesTheValueTwiceLittleEndian(t *testing.T) {
	f := &fake{answers: []uint16{uint16(StatusOK)}}
	g := &Glasses{p: f}
	if err := g.Set(0x24, 0x32); err != nil {
		t.Fatal(err)
	}
	got := f.asked[0]
	want := []byte{0x10, 0x00, 0x24, viture.DirWrite, 0x02, 0x00, 0x32, 0x00, 0x32, 0x00}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the frame is % x, want % x", got[:10], want)
		}
	}
	if len(got) != viture.ReportSize {
		t.Errorf("the report is %d bytes, want %d", len(got), viture.ReportSize)
	}
}

// TestAReadCarriesNoValueAtAll, or it is a command.
func TestAReadCarriesNoValueAtAll(t *testing.T) {
	f := &fake{answers: []uint16{7}}
	g := &Glasses{p: f}
	v, err := g.Get(0x22)
	if err != nil {
		t.Fatal(err)
	}
	if v != 7 {
		t.Errorf("Get = %d", v)
	}
	got := f.asked[0]
	if got[3] != viture.DirRead {
		t.Errorf("the read went out as direction %#02x", got[3])
	}
	for i := 6; i < 10; i++ {
		if got[i] != 0 {
			t.Fatalf("a read carries a value: % x", got[6:10])
		}
	}
}

// TestTwoMeansThereIsNoSuchSetting.
//
// ⭐ AND THAT IS WHAT MAKES THE WHOLE ID SPACE DISCOVERABLE WITHOUT WRITING:
// twenty messages exist between 0x00 and 0x7f and every other id answers 2, so
// a sweep of reads maps the device and changes nothing.
func TestTwoMeansThereIsNoSuchSetting(t *testing.T) {
	g := &Glasses{p: &fake{answers: []uint16{2}}}
	if _, err := g.Get(0x55); !errors.Is(err, ErrNoSetting) {
		t.Errorf("a 2 came back as %v", err)
	}
}

// TestTheHeadsetsOwnRefusalIsReported, in its own words.
func TestTheHeadsetsOwnRefusalIsReported(t *testing.T) {
	for _, c := range []struct {
		v    uint16
		says string
	}{
		{uint16(StatusRefused), "refused"},
		{uint16(StatusTooShort), "too short"},
		{99, "code 99"},
	} {
		g := &Glasses{p: &fake{answers: []uint16{c.v}}}
		err := g.Set(0x24, 1)
		if err == nil {
			t.Fatalf("status %d passed as success", c.v)
		}
		if got := err.Error(); got != "beast: the headset "+c.says {
			t.Errorf("status %d reads %q", c.v, got)
		}
	}
	if err := Status(StatusOK).Err(); err != nil {
		t.Errorf("taken reads as %v", err)
	}
	if got := StatusOK.String(); got != "taken" {
		t.Errorf("StatusOK reads %q", got)
	}
}

// TestNudgeReadsBeforeItWrites, and clamps.
//
// ⛔ A KEY THAT MEANS "BRIGHTER" HAS TO KNOW WHAT IT IS BRIGHTER THAN, and the
// headset is the only thing that knows: somebody may have used the buttons on
// the arm since anything here last looked.
func TestNudgeReadsBeforeItWrites(t *testing.T) {
	f := &fake{answers: []uint16{4, uint16(StatusOK)}}
	g := &Glasses{p: f}
	at, err := g.Nudge(0x22, +1, 8)
	if err != nil || at != 5 {
		t.Fatalf("Nudge = %d, %v", at, err)
	}
	if f.dirs[0] != viture.DirRead || f.dirs[1] != viture.DirWrite {
		t.Errorf("it went %v, want a read then a write", f.dirs)
	}

	// At the top it does not write at all: a write that changes nothing is a
	// round trip to a microcontroller for no reason.
	f = &fake{answers: []uint16{8}}
	g = &Glasses{p: f}
	if at, err := g.Nudge(0x22, +1, 8); err != nil || at != 8 {
		t.Errorf("at the top Nudge = %d, %v", at, err)
	}
	if len(f.asked) != 1 {
		t.Errorf("it made %d exchanges at the top, want 1", len(f.asked))
	}
	// And at the bottom, clamped rather than wrapped.
	f = &fake{answers: []uint16{0}}
	if at, err := (&Glasses{p: f}).Nudge(0x22, -1, 8); err != nil || at != 0 {
		t.Errorf("at the bottom Nudge = %d, %v", at, err)
	}
}

// TestEveryWayAnExchangeCanFail, and each says which.
func TestEveryWayAnExchangeCanFail(t *testing.T) {
	var none *Glasses
	if _, err := none.Get(0x22); !errors.Is(err, ErrNoDevice) {
		t.Errorf("reading nothing = %v", err)
	}
	if err := none.Set(0x22, 1); !errors.Is(err, ErrNoDevice) {
		t.Errorf("writing nothing = %v", err)
	}
	if err := none.Close(); err != nil {
		t.Errorf("closing nothing = %v", err)
	}
	if _, err := none.Nudge(0x22, 1, 8); !errors.Is(err, ErrNoDevice) {
		t.Errorf("nudging nothing = %v", err)
	}

	silent := &Glasses{p: &fake{err: ErrNoAnswer}}
	if _, err := silent.Get(0x22); !errors.Is(err, ErrNoAnswer) {
		t.Errorf("a silent headset = %v", err)
	}
	if err := silent.Set(0x22, 1); !errors.Is(err, ErrNoAnswer) {
		t.Errorf("a silent headset = %v", err)
	}
	if _, err := (&Glasses{p: &fake{answers: []uint16{4}, err: nil}}).Nudge(0x22, 1, 8); err == nil {
		t.Error("a refused write inside Nudge passed")
	}
	if err := (&Glasses{p: &fake{}}).Close(); err != nil {
		t.Errorf("Close = %v", err)
	}
}

// TestOpenReportsWhatThePlatformSaid, on every operating system.
func TestOpenReportsWhatThePlatformSaid(t *testing.T) {
	was := openDevice
	t.Cleanup(func() { openDevice = was })
	openDevice = func() (*Glasses, error) { return nil, ErrNoDevice }
	if _, err := Open(); !errors.Is(err, ErrNoDevice) {
		t.Errorf("Open with nothing attached = %v", err)
	}
	want := &Glasses{p: &fake{}}
	openDevice = func() (*Glasses, error) { return want, nil }
	if got, err := Open(); err != nil || got != want {
		t.Errorf("Open() = %v, %v", got, err)
	}
}
