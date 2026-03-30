package tty

import (
	"golang.org/x/sys/unix"
	"fmt"
)

type Tty struct {
	ws *unix.Winsize
}

const (
	ALT_SCREEN_BUFF = "\x1b[?1049h"
	DALT_SCREEN_BUFF = "\x1b[?1049l"
	MOV_CUR_POS = "\x1b[%d;%dH"
)

func NewTtyConfig() (*Tty, error) {
	ws, err := unix.IoctlGetWinsize(unix.Stdout, unix.TIOCGWINSZ)

	if err != nil {
		return nil, err
	}

	return &Tty{
		ws: ws,
	}, nil
}

func (tty *Tty) AlternateScreenBuffer() {
	fmt.Print(ALT_SCREEN_BUFF)
}

func (tty *Tty) MoveCurPos() {
	fmt.Printf(MOV_CUR_POS, tty.ws.Row, tty.ws.Col)
}

func (tty *Tty) InitialTtyPrompt() {
	tty.AlternateScreenBuffer()
	tty.MoveCurPos()
}

func (tty *Tty) ShutdownTtyPrompt() {
	fmt.Print(DALT_SCREEN_BUFF)
}
