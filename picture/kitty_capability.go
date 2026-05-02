package picture

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// KittyCapability reports whether the host terminal supports the Kitty
// graphics protocol. The state is process-wide — terminal support is a
// property of the tty, not of any individual Model — and shared by every
// picture.Model, chartpicture.Model, heatpicture.Model, and
// pictureurl.Model in the program.
type KittyCapability int8

const (
	// KittyCapabilityUnknown is the default state, before the probe has
	// completed. Toggle into Kitty is allowed in this state — Kitty
	// terminals usually respond within a few milliseconds, so the
	// Unknown window is brief.
	KittyCapabilityUnknown KittyCapability = iota

	// KittyCapabilitySupported means the terminal answered the Kitty
	// query — Kitty graphics escapes will render correctly.
	KittyCapabilitySupported

	// KittyCapabilityUnsupported means the probe timed out without a
	// response. Toggle into Kitty mode no-ops to avoid emitting Kitty
	// escapes that would print as garbage.
	KittyCapabilityUnsupported
)

// kittyProbeID is the image ID used for the Kitty support probe.
// Chosen to be far above any sensible consumer-assigned ID so it can
// never collide with real images.
const kittyProbeID = 42069101

// kittyProbeTimeout is how long QueryKittySupport waits for a response
// before concluding the terminal does not support Kitty graphics.
// Kitty-supporting terminals typically respond well under 50ms; 250ms
// is generous enough to accommodate slow ssh paths.
const kittyProbeTimeout = 250 * time.Millisecond

var (
	kittyCap       atomic.Int32 // holds KittyCapability values
	kittyQueryOnce sync.Once
)

// KittySupported reports the current process-wide Kitty graphics
// capability. Returns KittyCapabilityUnknown until the probe started by
// QueryKittySupport resolves (typically <50ms after the first Init).
func KittySupported() KittyCapability {
	return KittyCapability(kittyCap.Load())
}

// ForceKittyCapability sets the process-wide Kitty graphics capability,
// bypassing terminal probing. **Typically used in tests** — production
// code should rely on QueryKittySupport batched from Model.Init. May
// also be useful in transports where auto-detection misfires (some tmux
// passthrough setups, terminal multiplexer chains) and the application
// has out-of-band knowledge of true terminal support.
func ForceKittyCapability(c KittyCapability) {
	kittyCap.Store(int32(c))
}

// kittyProbeTickMsg fires kittyProbeTimeout after QueryKittySupport
// runs; if the capability is still Unknown when Model.Update sees this,
// it concludes Kitty is unsupported.
type kittyProbeTickMsg struct{}

// QueryKittySupport returns a Cmd that probes terminal Kitty graphics
// support. The probe runs at most once per process via sync.Once;
// subsequent calls return nil. Multiple Models can safely batch this
// from their Init — only the first emission actually queries the
// terminal.
//
// Two messages drive resolution:
//   - uv.KittyGraphicsEvent with the probe's ID arrives if the terminal
//     supports the protocol; Model.Update sets the capability to
//     KittyCapabilitySupported.
//   - kittyProbeTickMsg fires after kittyProbeTimeout; if the
//     capability is still Unknown, Model.Update sets it to
//     KittyCapabilityUnsupported.
//
// Both messages are intercepted by Model.Update — consumers that
// forward every tea.Msg to Model.Update don't need to handle them.
func QueryKittySupport() tea.Cmd {
	var cmd tea.Cmd
	kittyQueryOnce.Do(func() {
		cmd = tea.Batch(
			tea.Raw(buildKittyQueryAPC(kittyProbeID)),
			tea.Tick(kittyProbeTimeout, func(time.Time) tea.Msg {
				return kittyProbeTickMsg{}
			}),
		)
	})
	return cmd
}

// buildKittyQueryAPC encodes a Kitty graphics query (a=q): a tiny 1×1
// transmit whose only purpose is to elicit a response. Kitty terminals
// reply with `\e_Gi=<id>;OK\e\\`; non-Kitty terminals don't reply.
// The payload "AAAA" is base64 of three zero bytes (one RGB pixel).
func buildKittyQueryAPC(id int) string {
	return fmt.Sprintf("\x1b_Ga=q,t=d,f=24,s=1,v=1,i=%d;AAAA\x1b\\", id)
}

// recordKittyResponse handles a uv.KittyGraphicsEvent. Any response
// carrying our probe's image ID proves the terminal speaks the protocol
// (even an error response — only a Kitty-aware terminal would have
// produced it). CompareAndSwap so a Forced capability or an earlier
// response wins over a late one.
func recordKittyResponse(ev uv.KittyGraphicsEvent) {
	if ev.Options.ID != kittyProbeID {
		return
	}
	kittyCap.CompareAndSwap(int32(KittyCapabilityUnknown), int32(KittyCapabilitySupported))
}

// recordKittyTimeout marks Kitty unsupported if the probe window has
// elapsed without a response. Idempotent and a no-op if the capability
// was already resolved (by an earlier response or by ForceKittyCapability).
func recordKittyTimeout() {
	kittyCap.CompareAndSwap(int32(KittyCapabilityUnknown), int32(KittyCapabilityUnsupported))
}
