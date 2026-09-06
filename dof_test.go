// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package beast

import (
	"strings"
	"testing"

	"github.com/go-macos/iokit/viture"
)

// ⛔⛔ THE MESSAGE IS 0x43, AND THE TEST IS THAT IT IS NOT 0x44. What the
// headset ANNOUNCES and what it is TOLD are numbered differently and collide:
// 0x44 announces the tracking mode and is WRITTEN to set the SIDE mode. Writing
// there to anchor a picture makes it small and puts it bottom-left with nothing
// anchored -- which is what happened on a real desk before this was understood.
func TestAnchoringWritesTheTrackingByteAndNotTheSideMode(t *testing.T) {
	f := &fake{answers: []uint16{uint16(StatusOK)}}
	g := &Glasses{p: f}
	if err := g.SetDOF(DOF3); err != nil {
		t.Fatalf("anchor: %v", err)
	}
	if len(f.asked) != 1 {
		t.Fatalf("%d reports sent, want 1", len(f.asked))
	}
	msg := f.asked[0][2]
	if msg != viture.CmdNativeDOF {
		t.Errorf("anchoring wrote message %#x, want %#x", msg, viture.CmdNativeDOF)
	}
	if msg == viture.MsgNativeDOF {
		t.Errorf("anchoring wrote %#x, which is what the headset ANNOUNCES the "+
			"tracking mode on and what it is TOLD the SIDE mode on", msg)
	}
}

func TestReadingTheModeAsksTheSameByte(t *testing.T) {
	f := &fake{answers: []uint16{2}}
	g := &Glasses{p: f}
	got, err := g.DOF()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != DOFSmoothFollow {
		t.Errorf("read %v, want smooth follow", got)
	}
	if f.asked[0][2] != viture.CmdNativeDOF {
		t.Errorf("read message %#x, want %#x", f.asked[0][2], viture.CmdNativeDOF)
	}
}

// ⛔ RECENTRING IS ITS OWN COMMAND. Anchoring without it leaves the picture
// wherever the head happened to be pointing.
func TestRecentringWritesZeroToItsOwnByte(t *testing.T) {
	f := &fake{answers: []uint16{0}}
	g := &Glasses{p: f}
	if err := g.Recenter(); err != nil {
		t.Fatalf("recentre: %v", err)
	}
	if got := f.asked[0][2]; got != viture.CmdNativeRecenter {
		t.Errorf("recentring wrote %#x, want %#x", got, viture.CmdNativeRecenter)
	}
}

// ⛔ THERE IS NO 6DOF ON THIS HARDWARE, and a caller asking for one gets a
// refusal rather than a value the headset would have to interpret.
func TestAFourthModeIsRefusedRatherThanSent(t *testing.T) {
	f := &fake{answers: []uint16{0}}
	g := &Glasses{p: f}
	err := g.SetDOF(DOF(6))
	if err == nil {
		t.Fatal("a mode the headset does not have was sent to it")
	}
	if !strings.Contains(err.Error(), "three") {
		t.Errorf("the refusal should say how many there are, got %q", err)
	}
	if len(f.asked) != 0 {
		t.Errorf("%d reports were sent anyway", len(f.asked))
	}
}

func TestEachModeSaysWhatItDoes(t *testing.T) {
	for d, want := range map[DOF]string{
		DOFNone: "fixed", DOF3: "anchored", DOFSmoothFollow: "follow", DOF(9): "does not name",
	} {
		if got := d.String(); !strings.Contains(got, want) {
			t.Errorf("DOF(%d) says %q, want something containing %q", uint16(d), got, want)
		}
	}
}

// Bypass is the precondition nobody sees: a headset showing the host's video as
// it arrives has nothing to anchor.
func TestNativeSaysWhetherThereIsAnythingToAnchor(t *testing.T) {
	for v, want := range map[uint16]bool{0: false, 1: true} {
		f := &fake{answers: []uint16{v}}
		g := &Glasses{p: f}
		got, err := g.Native()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if got != want {
			t.Errorf("native mode %d read as %v", v, got)
		}
		if f.asked[0][2] != viture.CmdNativeMode {
			t.Errorf("asked %#x, want %#x", f.asked[0][2], viture.CmdNativeMode)
		}
	}
}

// ⛔⛔ THE MIDDLE MODE IS THE ONE THE GENERAL READER CANNOT READ. A read reply
// carrying 2 is ambiguous on this protocol -- the value two, or the status "no
// such message" -- and Get resolves it as the refusal, which is the only thing
// it can do in general. Smooth follow IS two.
//
// So a three-state menu built on Get would report "this headset has no tracking
// setting" at the exact moment the headset is in the middle of the three.
func TestSmoothFollowIsReadableEvenThoughTwoMeansNoSuchSetting(t *testing.T) {
	// What the general reader does with the same answer.
	if _, err := (&Glasses{p: &fake{answers: []uint16{2}}}).Get(viture.CmdNativeDOF); err == nil {
		t.Fatal("Get no longer treats 2 as a refusal; this test's premise is gone")
	} else if !strings.Contains(err.Error(), "no such setting") {
		t.Fatalf("Get answered %q, and this test is about the refusal", err)
	}
	// What this one does.
	got, err := (&Glasses{p: &fake{answers: []uint16{2}}}).DOF()
	if err != nil {
		t.Fatalf("smooth follow could not be read: %v", err)
	}
	if got != DOFSmoothFollow {
		t.Errorf("read %v, want smooth follow", got)
	}
}

// A value outside the three is reported rather than silently becoming a mode
// this package invented a name for.
func TestAnAnswerOutsideTheThreeIsReported(t *testing.T) {
	_, err := (&Glasses{p: &fake{answers: []uint16{7}}}).DOF()
	if err == nil {
		t.Fatal("7 was accepted as a tracking mode")
	}
	if !strings.Contains(err.Error(), "7") {
		t.Errorf("the error should carry what was answered, got %q", err)
	}
}
