//go:build linux

// Package render provides X11 window hint support for skip_taskbar and skip_pager.
package render

import (
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// WindowHintApplier handles applying X11 EWMH hints to windows.
// It caches the X11 connection and atoms for efficiency.
type WindowHintApplier struct {
	mu       sync.Mutex
	conn     *xgb.Conn
	atoms    map[string]xproto.Atom
	initDone bool
}

// globalHintApplier is a singleton for applying window hints.
var globalHintApplier = &WindowHintApplier{
	atoms: make(map[string]xproto.Atom),
}

// ApplyWindowHints sets X11 EWMH hints for skip_taskbar and skip_pager.
// This must be called after the window is created (e.g., after ebiten.RunGame starts).
// Returns nil on success or if hints are not applicable (e.g., non-X11 environment).
func ApplyWindowHints(skipTaskbar, skipPager bool) error {
	if !skipTaskbar && !skipPager {
		return nil
	}
	return globalHintApplier.Apply(skipTaskbar, skipPager)
}

// Apply sets the requested EWMH hints on the focused X11 window.
func (h *WindowHintApplier) Apply(skipTaskbar, skipPager bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Initialize connection if needed
	if err := h.ensureInit(); err != nil {
		return nil // Silently ignore if X11 is not available
	}

	// Get the active window
	window, err := h.getActiveWindow()
	if err != nil || window == xproto.WindowNone {
		return nil // Silently ignore if we can't get the window
	}

	// Build the list of atoms to add to _NET_WM_STATE
	var atoms []xproto.Atom

	if skipTaskbar {
		atom, err := h.getAtom("_NET_WM_STATE_SKIP_TASKBAR")
		if err == nil {
			atoms = append(atoms, atom)
		}
	}

	if skipPager {
		atom, err := h.getAtom("_NET_WM_STATE_SKIP_PAGER")
		if err == nil {
			atoms = append(atoms, atom)
		}
	}

	if len(atoms) == 0 {
		return nil
	}

	// Get the _NET_WM_STATE atom
	stateAtom, err := h.getAtom("_NET_WM_STATE")
	if err != nil {
		return nil
	}

	// Get current _NET_WM_STATE values
	currentAtoms, err := h.getWindowState(window, stateAtom)
	if err != nil {
		currentAtoms = []xproto.Atom{}
	}

	// Merge new atoms with existing ones (avoid duplicates)
	atomSet := make(map[xproto.Atom]bool)
	for _, a := range currentAtoms {
		atomSet[a] = true
	}
	for _, a := range atoms {
		atomSet[a] = true
	}

	// Convert back to slice
	finalAtoms := make([]xproto.Atom, 0, len(atomSet))
	for a := range atomSet {
		finalAtoms = append(finalAtoms, a)
	}

	// Set the _NET_WM_STATE property
	atomAtom, err := h.getAtom("ATOM")
	if err != nil {
		return nil
	}

	data := make([]byte, len(finalAtoms)*4)
	for i, a := range finalAtoms {
		xgb.Put32(data[i*4:], uint32(a))
	}

	xproto.ChangeProperty(h.conn, xproto.PropModeReplace, window,
		stateAtom, atomAtom, 32, uint32(len(finalAtoms)), data)

	return nil
}

// ensureInit initializes the X11 connection if not already done.
func (h *WindowHintApplier) ensureInit() error {
	if h.initDone {
		return nil
	}

	conn, err := xgb.NewConn()
	if err != nil {
		return err
	}

	h.conn = conn
	h.initDone = true
	return nil
}

// getAtom retrieves or interns an X11 atom by name.
func (h *WindowHintApplier) getAtom(name string) (xproto.Atom, error) {
	if atom, ok := h.atoms[name]; ok {
		return atom, nil
	}

	reply, err := xproto.InternAtom(h.conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, err
	}

	h.atoms[name] = reply.Atom
	return reply.Atom, nil
}

// getActiveWindow returns the currently focused/active window.
func (h *WindowHintApplier) getActiveWindow() (xproto.Window, error) {
	setup := xproto.Setup(h.conn)
	if len(setup.Roots) == 0 {
		return xproto.WindowNone, nil
	}
	root := setup.Roots[0].Root

	// Try _NET_ACTIVE_WINDOW first (EWMH standard)
	activeAtom, err := h.getAtom("_NET_ACTIVE_WINDOW")
	if err == nil {
		reply, err := xproto.GetProperty(h.conn, false, root, activeAtom,
			xproto.AtomWindow, 0, 1).Reply()
		if err == nil && reply != nil && len(reply.Value) >= 4 {
			return xproto.Window(xgb.Get32(reply.Value)), nil
		}
	}

	// Fallback to input focus
	focusReply, err := xproto.GetInputFocus(h.conn).Reply()
	if err != nil {
		return xproto.WindowNone, err
	}

	return focusReply.Focus, nil
}

// getWindowState retrieves the current _NET_WM_STATE atoms from a window.
func (h *WindowHintApplier) getWindowState(window xproto.Window, stateAtom xproto.Atom) ([]xproto.Atom, error) {
	atomAtom, err := h.getAtom("ATOM")
	if err != nil {
		return nil, err
	}

	reply, err := xproto.GetProperty(h.conn, false, window, stateAtom,
		atomAtom, 0, 256).Reply()
	if err != nil || reply == nil {
		return nil, err
	}

	atoms := make([]xproto.Atom, 0, len(reply.Value)/4)
	for i := 0; i+4 <= len(reply.Value); i += 4 {
		atoms = append(atoms, xproto.Atom(xgb.Get32(reply.Value[i:])))
	}

	return atoms, nil
}

// Close releases the X11 connection.
func (h *WindowHintApplier) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.conn != nil {
		h.conn.Close()
		h.conn = nil
	}
	h.initDone = false
	h.atoms = make(map[string]xproto.Atom)
}

// CloseWindowHints releases resources used by the window hint applier.
func CloseWindowHints() {
	globalHintApplier.Close()
}
