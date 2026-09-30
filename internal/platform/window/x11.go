package window

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type X11WindowController struct {
	connection       *xgb.Conn
	root             xproto.Window
	motifHints       xproto.Atom
	activeWindow     xproto.Atom
	netWindowName    xproto.Atom
	legacyWindowName xproto.Atom
	netWindowPID     xproto.Atom
	netClientList    xproto.Atom
	mu               sync.RWMutex
	targetWindow     xproto.Window
}

func NewX11WindowController() (*X11WindowController, error) {
	connection, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("conectar ao X11 para controlar a janela: %w", err)
	}

	setup := xproto.Setup(connection)
	if setup == nil || len(setup.Roots) == 0 {
		connection.Close()
		return nil, errors.New("X11 nao informou uma tela raiz para a janela")
	}

	controller := &X11WindowController{
		connection: connection,
		root:       setup.DefaultScreen(connection).Root,
	}
	for name, target := range map[string]*xproto.Atom{
		"_MOTIF_WM_HINTS":    &controller.motifHints,
		"_NET_ACTIVE_WINDOW": &controller.activeWindow,
		"_NET_WM_NAME":       &controller.netWindowName,
		"WM_NAME":            &controller.legacyWindowName,
		"_NET_WM_PID":        &controller.netWindowPID,
		"_NET_CLIENT_LIST":   &controller.netClientList,
	} {
		atom, err := internAtom(connection, name)
		if err != nil {
			controller.Close()
			return nil, fmt.Errorf("obter atom X11 %s: %w", name, err)
		}
		*target = atom
	}

	return controller, nil
}

func (c *X11WindowController) Close() {
	if c != nil && c.connection != nil {
		c.connection.Close()
	}
}

func (c *X11WindowController) HideDecorations(title string) error {
	target, err := c.findWindowByTitle(title)
	if err != nil {
		return err
	}
	if target == xproto.Window(0) {
		return fmt.Errorf("janela X11 %q nao encontrada", title)
	}
	c.mu.Lock()
	c.targetWindow = target
	c.mu.Unlock()

	hints := motifNoDecorationHints()
	if err := xproto.ChangePropertyChecked(
		c.connection,
		xproto.PropModeReplace,
		target,
		c.motifHints,
		c.motifHints,
		32,
		uint32(len(hints)/4),
		hints,
	).Check(); err != nil {
		return fmt.Errorf("ocultar decoracoes da janela X11: %w", err)
	}

	return nil
}

func (c *X11WindowController) IsFocused(title string) (bool, error) {
	target := c.targetWindowID(title)
	if target == xproto.Window(0) {
		return false, fmt.Errorf("janela X11 %q ainda nao foi identificada", title)
	}

	active, err := c.activeWindowID()
	if err != nil {
		return false, err
	}
	if active == xproto.Window(0) {
		return false, nil
	}

	return active == target, nil
}

func (c *X11WindowController) targetWindowID(title string) xproto.Window {
	c.mu.RLock()
	target := c.targetWindow
	c.mu.RUnlock()
	if target != xproto.Window(0) {
		return target
	}

	target, err := c.findWindowByTitle(title)
	if err != nil {
		return 0
	}
	if target != xproto.Window(0) {
		c.mu.Lock()
		c.targetWindow = target
		c.mu.Unlock()
	}
	return target
}

func (c *X11WindowController) findWindowByTitle(title string) (xproto.Window, error) {
	clients, err := c.clientWindows()
	if err != nil {
		return 0, err
	}

	for _, candidate := range clients {
		pid, err := c.windowPID(candidate)
		if err != nil {
			return 0, err
		}
		if pid == uint32(os.Getpid()) {
			return candidate, nil
		}

		candidateTitle, err := c.windowTitle(candidate)
		if err != nil {
			return 0, err
		}
		if candidateTitle == title {
			return candidate, nil
		}
	}

	return 0, nil
}

func (c *X11WindowController) clientWindows() ([]xproto.Window, error) {
	reply, err := xproto.GetProperty(
		c.connection,
		false,
		c.root,
		c.netClientList,
		xproto.GetPropertyTypeAny,
		0,
		4096,
	).Reply()
	if err != nil {
		return nil, fmt.Errorf("consultar lista de clientes X11: %w", err)
	}
	if reply == nil || len(reply.Value) < 4 {
		tree, err := xproto.QueryTree(c.connection, c.root).Reply()
		if err != nil {
			return nil, fmt.Errorf("consultar arvore de janelas X11: %w", err)
		}
		return tree.Children, nil
	}

	return windowIDsFromProperty(reply.Value), nil
}

func windowIDsFromProperty(value []byte) []xproto.Window {
	count := len(value) / 4
	windows := make([]xproto.Window, 0, count)
	for index := 0; index < count; index++ {
		start := index * 4
		windows = append(windows, xproto.Window(binary.LittleEndian.Uint32(value[start:start+4])))
	}
	return windows
}

func (c *X11WindowController) windowPID(window xproto.Window) (uint32, error) {
	reply, err := xproto.GetProperty(
		c.connection,
		false,
		window,
		c.netWindowPID,
		xproto.GetPropertyTypeAny,
		0,
		1,
	).Reply()
	if err != nil {
		return 0, fmt.Errorf("consultar PID da janela X11: %w", err)
	}
	if reply == nil || len(reply.Value) < 4 {
		return 0, nil
	}

	return binary.LittleEndian.Uint32(reply.Value), nil
}

func (c *X11WindowController) activeWindowID() (xproto.Window, error) {
	reply, err := xproto.GetProperty(
		c.connection,
		false,
		c.root,
		c.activeWindow,
		xproto.GetPropertyTypeAny,
		0,
		1,
	).Reply()
	if err != nil {
		return 0, fmt.Errorf("consultar janela X11 em foco: %w", err)
	}
	if reply == nil || len(reply.Value) < 4 {
		return 0, nil
	}

	return xproto.Window(binary.LittleEndian.Uint32(reply.Value)), nil
}

func (c *X11WindowController) windowTitle(window xproto.Window) (string, error) {
	for _, property := range []xproto.Atom{c.netWindowName, c.legacyWindowName} {
		reply, err := xproto.GetProperty(
			c.connection,
			false,
			window,
			property,
			xproto.GetPropertyTypeAny,
			0,
			1024,
		).Reply()
		if err != nil {
			return "", fmt.Errorf("consultar titulo da janela X11: %w", err)
		}
		if reply != nil && len(reply.Value) > 0 {
			return strings.TrimRight(string(reply.Value), "\x00"), nil
		}
	}

	return "", nil
}

func internAtom(connection *xgb.Conn, name string) (xproto.Atom, error) {
	reply, err := xproto.InternAtom(connection, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, err
	}
	return reply.Atom, nil
}

func motifNoDecorationHints() []byte {
	hints := []uint32{2, 0, 0, 0, 0}
	encoded := make([]byte, len(hints)*4)
	for index, hint := range hints {
		binary.LittleEndian.PutUint32(encoded[index*4:], hint)
	}
	return encoded
}
