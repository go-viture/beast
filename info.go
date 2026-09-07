// Copyright (c) the go-viture authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package beast

import (
	"github.com/go-macos/iokit/viture"
)

// ⭐⭐ A FAMILY OF READS THIS PACKAGE COULD NOT REACH.
//
// [Glasses.Get] sends viture.DirRead, which is 0x31 -- and that byte is the
// HIGH half of a 16-bit message id, so every read this package could make was
// 0x31xx. The ids that answer with TEXT are 0x30xx: a neighbouring family, one
// nibble away, and unreachable through Get.
//
// The ids come from VITURE's own web updater, which carries an R6 path for this
// headset, and every one is a READ -- which is what makes them safe to send.
const (
	// DirText is the high half of the ids that answer with text.
	DirText byte = 0x30

	// MsgBoardSerial identifies the board, MsgPackageSerial the product, and
	// MsgFirmwareVersion is the version a person recognises.
	MsgBoardSerial     byte = 0x02
	MsgPackageSerial   byte = 0x05
	MsgFirmwareVersion byte = 0x03
)

// Info is what the headset says about itself.
type Info struct {
	// FirmwareVersion is the application firmware: "20.0.01.027_20260825" on
	// the headset this was written against.
	FirmwareVersion string
	// BoardSerial identifies the board. PackageSerial identifies the product
	// and is EMPTY on a headset that has none -- this one answers 0xff repeated
	// for it, which is the vendor's own way of saying absent.
	BoardSerial, PackageSerial string
}

// text asks one of the text ids and returns what it answered.
func (g *Glasses) text(msg byte) (string, error) {
	if g == nil {
		return "", ErrNoDevice
	}
	e, err := g.p.exchange(msg, DirText, textCommand(msg))
	if err != nil {
		return "", err
	}
	return e.Text, nil
}

// Info asks the headset for its firmware version and serials.
//
// ⛔ A FIELD IT COULD NOT READ STAYS EMPTY. There is no second number to fall
// back on and no reason to invent one: a wrong version on a screen is worse
// than a blank, because nobody can tell it is wrong.
func (g *Glasses) Info() (Info, error) {
	var in Info
	var first error
	keep := func(dst *string, msg byte) {
		s, err := g.text(msg)
		if err != nil {
			if first == nil {
				first = err
			}
			return
		}
		*dst = s
	}
	keep(&in.FirmwareVersion, MsgFirmwareVersion)
	keep(&in.BoardSerial, MsgBoardSerial)
	keep(&in.PackageSerial, MsgPackageSerial)
	if in.FirmwareVersion == "" && in.BoardSerial == "" && in.PackageSerial == "" {
		return in, first
	}
	return in, nil
}

// textCommand builds a text read.
//
// ⛔ IT CANNOT GO THROUGH command(). That one decides the direction itself and
// always says DirRead -- 0x31 -- which is the wrong half of the wrong id here.
// And its length is 0x03, while these carry NONE: the frame that actually
// answered on the hardware was
//
//	10 00 <msg> 30 00 00 00 00
//
// eight bytes and a length of zero. Transcribed from the exchange that worked,
// not adapted from the one beside it.
func textCommand(msg byte) []byte {
	b := make([]byte, viture.ReportSize)
	copy(b, []byte{viture.EventHeader, 0x00, msg, DirText, 0x00, 0x00, 0x00, 0x00})
	return b
}
