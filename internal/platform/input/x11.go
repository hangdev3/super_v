package input

import (
	"errors"
	"fmt"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgb/xtest"
)

const (
	keysymControlLeft  xproto.Keysym = 0xffe3
	keysymControlRight xproto.Keysym = 0xffe4
	keysymVLower       xproto.Keysym = 0x0076
	keysymVUpper       xproto.Keysym = 0x0056
)

type X11Paster struct {
	connection *xgb.Conn
	root       xproto.Window
	control    xproto.Keycode
	v          xproto.Keycode
}

func NewX11Paster() (*X11Paster, error) {
	connection, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("conectar ao X11 para auto-colagem: %w", err)
	}

	if err := xtest.Init(connection); err != nil {
		connection.Close()
		return nil, fmt.Errorf("inicializar XTEST para auto-colagem: %w", err)
	}

	setup := xproto.Setup(connection)
	if setup == nil || len(setup.Roots) == 0 {
		connection.Close()
		return nil, errors.New("X11 nao informou uma tela raiz para auto-colagem")
	}

	root := setup.DefaultScreen(connection).Root
	keysyms, perKeycode, err := keyboardMapping(connection, setup.MinKeycode, setup.MaxKeycode)
	if err != nil {
		connection.Close()
		return nil, err
	}

	control, found := keycodeForKeysym(setup.MinKeycode, keysyms, perKeycode, keysymControlLeft, keysymControlRight)
	if !found {
		connection.Close()
		return nil, errors.New("tecla Control nao encontrada no mapa X11")
	}
	v, found := keycodeForKeysym(setup.MinKeycode, keysyms, perKeycode, keysymVLower, keysymVUpper)
	if !found {
		connection.Close()
		return nil, errors.New("tecla V nao encontrada no mapa X11")
	}

	return &X11Paster{
		connection: connection,
		root:       root,
		control:    control,
		v:          v,
	}, nil
}

func (p *X11Paster) Close() {
	if p != nil && p.connection != nil {
		p.connection.Close()
	}
}

func (p *X11Paster) Paste() error {
	if p == nil || p.connection == nil {
		return errors.New("auto-colagem X11 indisponivel")
	}

	if err := p.sendKey(xproto.KeyPress, p.control); err != nil {
		return err
	}
	if err := p.sendKey(xproto.KeyPress, p.v); err != nil {
		return err
	}
	if err := p.sendKey(xproto.KeyRelease, p.v); err != nil {
		return err
	}
	return p.sendKey(xproto.KeyRelease, p.control)
}

func (p *X11Paster) sendKey(eventType byte, keycode xproto.Keycode) error {
	if err := xtest.FakeInputChecked(
		p.connection,
		eventType,
		byte(keycode),
		0,
		p.root,
		0,
		0,
		0,
	).Check(); err != nil {
		return fmt.Errorf("enviar tecla X11: %w", err)
	}
	return nil
}

func keyboardMapping(connection *xgb.Conn, first, last xproto.Keycode) ([]xproto.Keysym, byte, error) {
	count := byte(last-first) + 1
	reply, err := xproto.GetKeyboardMapping(connection, first, count).Reply()
	if err != nil {
		return nil, 0, fmt.Errorf("consultar mapa de teclado X11: %w", err)
	}
	if reply == nil || reply.KeysymsPerKeycode == 0 {
		return nil, 0, errors.New("X11 retornou um mapa de teclado vazio")
	}
	return reply.Keysyms, reply.KeysymsPerKeycode, nil
}

func keycodeForKeysym(first xproto.Keycode, keysyms []xproto.Keysym, perKeycode byte, candidates ...xproto.Keysym) (xproto.Keycode, bool) {
	if perKeycode == 0 {
		return 0, false
	}

	wanted := make(map[xproto.Keysym]struct{}, len(candidates))
	for _, candidate := range candidates {
		wanted[candidate] = struct{}{}
	}

	for index, keysym := range keysyms {
		if _, ok := wanted[keysym]; !ok {
			continue
		}
		return first + xproto.Keycode(index/int(perKeycode)), true
	}
	return 0, false
}
