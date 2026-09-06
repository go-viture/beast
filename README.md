# go-viture/beast

Talk to a **VITURE Beast** headset from pure Go, `CGO_ENABLED=0`.

```go
g, err := beast.Open()
if err != nil { return err }
defer g.Close()

mode, err := g.Get(viture.MsgNativeDisplayMode)   // 0x31 = 1920x1080 @60Hz
err = g.Set(viture.MsgDisplayMode, viture.Mode3840x1080At60)  // side by side
at, err := g.Nudge(0x22, +1, 8)                   // one step brighter
```

## ⛔ Not the same protocol as the Luma

A Beast frame begins `0x10`; a Luma frame begins `FA 55`. They do not share an
envelope, a transport, or a habit. What they share is a house style: **ask
before you write, and judge by what the device answers.**

## ⭐ The headset answers every command with a status

```
0  taken        4  refused        6  too short
```

That is worth more than watching the display: **a screen that does not change
cannot tell a refusal from a command that never arrived.** Finding that out
took nineteen failed writes, all judged by a screen.

⭐ And a READ answering `2` means *"there is no such message"* — which makes the
whole id space discoverable **without ever writing**. Twenty messages exist
between `0x00` and `0x7f`; every other id answers 2.

## ⛔ The value goes in twice, little-endian

```
10 00 24 01 02 00 32 00 32 00        ← side by side
                ^^^^^ ^^^^^
```

That one detail is what nineteen attempts had wrong. The headset's own
**replies** carry a value little-endian and then big-endian, and copying that
shape into a command has it refused. **A reply and a command are not the same
frame.**

## ⚠ A setting can be there and then not be

Measured: `0x22` (brightness) and `0x30` (volume) both answered a read one
evening and were both gone ninety minutes later — same cable, no replug, every
other id still answering. `0x22` came back the next morning and was gone again
by lunchtime.

So `ErrNoSetting` is a **state this reports**, never a fact compiled in.

## Where the frames come from

`go-macos/iokit/viture`, which decoded them — not copied here. This package is
the transport and the ergonomics: open the right interface, **listen before
asking**, match the reply on message *and* direction, and hand back what the
headset said.

⛔ **Listen before asking**: the headset answers in milliseconds, and a listener
started afterwards misses its own answer.

## Requirements

macOS for the transport; the frame and exchange logic builds and is tested
everywhere.

## Licence

BSD-3-Clause.
