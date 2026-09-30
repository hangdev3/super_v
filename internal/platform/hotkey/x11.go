package hotkey

import (
	"errors"
	"fmt"
	"sync"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type Event struct {
	Timestamp uint32
}

type Modifier uint16

type SuperV struct {
	connection        *xgb.Conn
	root              xproto.Window
	keycode           xproto.Keycode
	keysym            xproto.Keysym
	requiredModifiers Modifier
	modifierMasks     []Modifier
	displayName       string
	events            chan Event
	stop              chan struct{}
	listenerWG        sync.WaitGroup
	closeOnce         sync.Once
	registered        bool
}

func NewSuperV() *SuperV {
	return newShortcut("Super+V", xproto.Keysym(0x0076), x11Mod4Mask)
}

func NewSuperShiftV() *SuperV {
	return newShortcut("Super+Shift+V", xproto.Keysym(0x0076), x11Mod4Mask|x11ShiftMask)
}

func NewSuperShiftS() *SuperV {
	return newShortcut("Super+Shift+S", xproto.Keysym(0x0073), x11Mod4Mask|x11ShiftMask)
}

func newShortcut(displayName string, keysym xproto.Keysym, requiredModifiers Modifier) *SuperV {
	return &SuperV{
		keysym:            keysym,
		requiredModifiers: requiredModifiers,
		modifierMasks:     shortcutModifierMasks(requiredModifiers),
		displayName:       displayName,
		events:            make(chan Event),
		stop:              make(chan struct{}),
	}
}

func (h *SuperV) Register() error {
	if h.registered {
		return fmt.Errorf("%s ja foi registrado", h.displayName)
	}

	connection, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("conectar ao X11 para registrar %s: %w", h.displayName, err)
	}

	setup := xproto.Setup(connection)
	if setup == nil || len(setup.Roots) == 0 {
		connection.Close()
		return fmt.Errorf("X11 nao informou uma tela raiz para %s", h.displayName)
	}

	keycode, err := findKeycode(connection, setup.MinKeycode, setup.MaxKeycode, h.keysym)
	if err != nil {
		connection.Close()
		return err
	}

	root := setup.DefaultScreen(connection).Root
	registeredMasks := make([]Modifier, 0, len(h.modifierMasks))
	for _, modifiers := range h.modifierMasks {
		if err := xproto.GrabKeyChecked(
			connection,
			false,
			root,
			uint16(modifiers),
			keycode,
			xproto.GrabModeAsync,
			xproto.GrabModeAsync,
		).Check(); err != nil {
			for _, registeredModifiers := range registeredMasks {
				_ = xproto.UngrabKeyChecked(connection, keycode, root, uint16(registeredModifiers)).Check()
			}
			connection.Close()
			return fmt.Errorf("registrar grab X11 para %s: %w", h.displayName, err)
		}
		registeredMasks = append(registeredMasks, modifiers)
	}
	h.connection = connection
	h.root = root
	h.keycode = keycode
	h.registered = true
	h.listenerWG.Add(1)
	go h.listen()
	return nil
}

func (h *SuperV) Events() <-chan Event {
	return h.events
}

func (h *SuperV) Close() error {
	var firstErr error
	h.closeOnce.Do(func() {
		close(h.stop)
		if h.connection != nil {
			for _, modifiers := range h.modifierMasks {
				if err := xproto.UngrabKeyChecked(h.connection, h.keycode, h.root, uint16(modifiers)).Check(); err != nil && firstErr == nil {
					firstErr = err
				}
			}
			h.connection.Close()
		}
	})
	h.listenerWG.Wait()
	return firstErr
}

const (
	// X11's LockMask is set when Caps Lock is active. Mod2 is commonly Num Lock.
	x11ShiftMask Modifier = 1 << 0
	x11LockMask  Modifier = 1 << 1
	x11Mod4Mask  Modifier = 1 << 6
	x11Mod2Mask  Modifier = 1 << 4
	x11Mod3Mask  Modifier = 1 << 5
)

func superVModifierMasks() []Modifier {
	return shortcutModifierMasks(x11Mod4Mask)
}

func shortcutModifierMasks(required Modifier) []Modifier {
	lockMasks := []Modifier{x11LockMask, x11Mod2Mask, x11Mod3Mask}
	masks := make([]Modifier, 0, 1<<len(lockMasks))
	for combination := 0; combination < 1<<len(lockMasks); combination++ {
		modifiers := required
		for bit, lockMask := range lockMasks {
			if combination&(1<<bit) != 0 {
				modifiers |= lockMask
			}
		}
		masks = append(masks, modifiers)
	}
	return masks
}

func (h *SuperV) listen() {
	defer h.listenerWG.Done()
	defer close(h.events)

	for {
		x11Event, x11Error := h.connection.WaitForEvent()
		if x11Event == nil && x11Error == nil {
			return
		}
		if x11Error != nil {
			return
		}
		event, ok := shortcutEvent(x11Event, h.keycode)
		if !ok {
			continue
		}

		select {
		case h.events <- event:
		case <-h.stop:
			return
		}
	}
}

func isShortcutKeyPress(event xgb.Event, keycode xproto.Keycode) bool {
	_, ok := shortcutEvent(event, keycode)
	return ok
}

func shortcutEvent(event xgb.Event, keycode xproto.Keycode) (Event, bool) {
	switch typedEvent := event.(type) {
	case xproto.KeyPressEvent:
		if typedEvent.Detail != keycode {
			return Event{}, false
		}
		return Event{Timestamp: uint32(typedEvent.Time)}, true
	case *xproto.KeyPressEvent:
		if typedEvent == nil || typedEvent.Detail != keycode {
			return Event{}, false
		}
		return Event{Timestamp: uint32(typedEvent.Time)}, true
	default:
		return Event{}, false
	}
}

func findKeycode(connection *xgb.Conn, first, last xproto.Keycode, keysym xproto.Keysym) (xproto.Keycode, error) {
	count := byte(last-first) + 1
	reply, err := xproto.GetKeyboardMapping(connection, first, count).Reply()
	if err != nil {
		return 0, fmt.Errorf("consultar mapa de teclado X11: %w", err)
	}
	if reply == nil || reply.KeysymsPerKeycode == 0 {
		return 0, errors.New("X11 retornou um mapa de teclado vazio")
	}

	perKeycode := int(reply.KeysymsPerKeycode)
	for index, mappedKeysym := range reply.Keysyms {
		if mappedKeysym == keysym {
			return first + xproto.Keycode(index/perKeycode), nil
		}
	}
	return 0, errors.New("tecla solicitada nao encontrada no mapa de teclado X11")
}
