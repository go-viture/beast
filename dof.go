// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package beast

import (
	"fmt"

	"github.com/go-macos/iokit/viture"
)

// DOF is where the picture sits while the head moves.
//
// ⭐ THE GLASSES DO THIS THEMSELVES. Nothing is computed here and no camera is
// opened: the Beast tracks its own orientation and composites the host's video
// where the tracking says. That is the whole difference from the Luma Ultra,
// whose 6DOF is visual-inertial odometry run on the host from the cameras --
// and it is why this is a command rather than a project.
//
// ⛔ THERE IS NO 6DOF HERE, AND ASKING FOR ONE WOULD BE INVENTING IT. The
// manufacturer's public header lists exactly three native modes and no more.
// Position tracking lives in viture_device_carina.h, which is the Luma's.
type DOF uint16

const (
	// DOFNone is the picture fixed to the glasses: it goes where the head goes.
	DOFNone DOF = 0
	// DOF3 anchors the picture in space. Turn your head and it stays put.
	DOF3 DOF = 1
	// DOFSmoothFollow lets the picture trail the head instead of being nailed
	// to either: it drifts back to centre rather than staying behind.
	DOFSmoothFollow DOF = 2
)

// String is what the mode is called, in a sentence somebody reads.
func (d DOF) String() string {
	switch d {
	case DOFNone:
		return "fixed to the glasses"
	case DOF3:
		return "anchored in space"
	case DOFSmoothFollow:
		return "smooth follow"
	}
	return fmt.Sprintf("mode %d, which this package does not name", uint16(d))
}

// DOF reads which of the three the glasses are in.
//
// ⛔⛔ THE MESSAGE IS 0x43 AND NOT 0x44, AND GETTING THAT WRONG REFRAMES
// SOMEBODY'S DESK. What the headset ANNOUNCES and what it is TOLD are numbered
// differently and collide: 0x44 announces the tracking mode and is WRITTEN to
// set the side mode. Writing 1 there to anchor a picture makes it small and
// puts it bottom-left, with nothing anchored -- measured, on a real desk.
// See [viture.CmdNativeDOF] for the three measurements that settled it.
// ⛔⛔ AND IT DOES NOT GO THROUGH [Glasses.Get], BECAUSE Get CANNOT READ ONE OF
// THE THREE MODES. A read reply carrying 2 is ambiguous on this protocol: it is
// either the value two or the status "no such message", and Get resolves that
// ambiguity the only way it can in general -- as the refusal. Smooth follow IS
// two. So the general reader reports the headset has no tracking setting at the
// exact moment it is in the middle one.
//
// It is resolved here because this message is not the general case: 0x43 was
// measured answering 0 and 1 on real hardware, so the setting exists and a 2 is
// the value. That reasoning does not transfer to any other message and is not
// pushed down into Get.
func (g *Glasses) DOF() (DOF, error) {
	if g == nil {
		return 0, ErrNoDevice
	}
	e, err := g.p.exchange(viture.CmdNativeDOF, viture.DirRead,
		command(viture.CmdNativeDOF, 0x03, 0))
	if err != nil {
		return 0, err
	}
	if e.Value > uint16(DOFSmoothFollow) {
		return 0, fmt.Errorf("beast: the headset answered %d for the tracking mode, "+
			"which is not one of the three it has", e.Value)
	}
	return DOF(e.Value), nil
}

// SetDOF puts the glasses in one of the three modes.
//
// ⚠ IT NEEDS THE GLASSES TO BE IN NATIVE MODE, and this does not put them
// there. The manufacturer's header says native tracking is refused outright
// while the device is in bypass -- showing the host's video as it arrives
// rather than compositing it. [Glasses.Native] reports which it is in.
func (g *Glasses) SetDOF(d DOF) error {
	if d > DOFSmoothFollow {
		return fmt.Errorf("beast: %d is not one of the three native modes", uint16(d))
	}
	return g.Set(viture.CmdNativeDOF, uint16(d))
}

// Recenter puts an anchored picture back in front of whoever is wearing the
// glasses.
//
// ⛔ IT IS A SEPARATE COMMAND, AND ANCHORING WITHOUT IT LEAVES THE PICTURE
// WHEREVER THE HEAD HAPPENED TO BE. The vendor library writes a payload of zero
// to 0x30, which is also the byte the headset announces its VOLUME on --
// another collision between the two numberings, and another reason the two
// lists are kept apart.
func (g *Glasses) Recenter() error { return g.Set(viture.CmdNativeRecenter, 0) }

// Native reports whether the glasses composite the picture themselves (true) or
// show the host's video as it arrives (false).
//
// ⭐ IT IS THE PRECONDITION FOR EVERYTHING ELSE IN THIS FILE. A headset in
// bypass has nothing to anchor: it is a monitor.
func (g *Glasses) Native() (bool, error) {
	v, err := g.Get(viture.CmdNativeMode)
	if err != nil {
		return false, err
	}
	return v == 1, nil
}
