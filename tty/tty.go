package tty

import (
	"context"
	"golang.org/x/sys/unix"
	"fmt"
)

type Tty struct {
	ws *unix.Winsize
	stdinTermios *unix.Termios
	stdoutTermios *unix.Termios
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

	stdinTermios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)

	if err != nil {
		return nil, err
	}

	stdoutTermios, err := unix.IoctlGetTermios(unix.Stdin, unix.TCGETS)

	if err != nil {
		return nil, err
	}

	return &Tty{
		ws: ws,
		stdinTermios: stdinTermios,
		stdoutTermios: stdoutTermios,
	}, nil
}

func (tty *Tty) AlternateScreenBuffer() {
	fmt.Print(ALT_SCREEN_BUFF)
}

func (tty *Tty) MoveCurPos(row, col uint16) {
	fmt.Printf(MOV_CUR_POS, row, col)
}

func (tty *Tty) InitialTtyPrompt() {
	tty.AlternateScreenBuffer()
	tty.MoveCurPos(tty.ws.Row, 0)
}

func (tty *Tty) ShutdownTtyPrompt(ctx context.Context) {
	go func() {
		<-ctx.Done()

		fmt.Print(DALT_SCREEN_BUFF)
		unix.Exit(0)
	}()
}

func (tty *Tty) EnableEcho() {
	tty.stdinTermios.Lflag |= unix.ECHO
	tty.stdoutTermios.Lflag |= unix.ECHO
	unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdoutTermios)
}

func (tty *Tty) DisableEcho() {
	tty.stdinTermios.Lflag &^= unix.ECHO
	tty.stdoutTermios.Lflag &^= unix.ECHO
	unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdoutTermios)
}
