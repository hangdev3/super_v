package pointer

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

type Position struct {
	X int
	Y int
}

type X11Pointer struct {
	connection *xgb.Conn
	root       xproto.Window
}

func NewX11Pointer() (*X11Pointer, error) {
	if err := validateX11Environment(os.Getenv("XDG_SESSION_TYPE"), os.Getenv("DISPLAY")); err != nil {
		return nil, err
	}

	connection, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("conectar ao X11: %w", err)
	}

	setup := xproto.Setup(connection)
	if setup == nil || len(setup.Roots) == 0 {
		connection.Close()
		return nil, errors.New("X11 nao informou uma tela raiz")
	}

	return &X11Pointer{
		connection: connection,
		root:       setup.DefaultScreen(connection).Root,
	}, nil
}

func (p *X11Pointer) Position() (Position, error) {
	reply, err := xproto.QueryPointer(p.connection, p.root).Reply()
	if err != nil {
		return Position{}, fmt.Errorf("consultar ponteiro X11: %w", err)
	}

	return positionFromReply(*reply), nil
}

func (p *X11Pointer) Close() {
	if p != nil && p.connection != nil {
		p.connection.Close()
	}
}

func positionFromReply(reply xproto.QueryPointerReply) Position {
	return Position{X: int(reply.RootX), Y: int(reply.RootY)}
}

func validateX11Environment(sessionType, display string) error {
	if sessionType == "wayland" {
		return errors.New("Wayland detectado: o MVP exige uma sessao X11/Cinnamon")
	}
	if display == "" {
		return errors.New("X11 indisponivel: a variavel DISPLAY nao esta definida")
	}

	return nil
}
