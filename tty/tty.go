package tty

import (
	// "context"
	"golang.org/x/sys/unix"
	"fmt"
	"os"
	"os/signal"
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
	ENABLE_SHOW_CURSOR = "\x1b[?25h"
	DISABLE_SHOW_CURSOR = "\x1b[?25l"
	ENABLE_BLINKING_CURSOR = "\x1b[?12h"
	DISABLE_BLINKING_CURSOR = "\x1b[?12l"
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
	tty.stdinTermios.Lflag &^= unix.ECHOCTL
	err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)

	if err != nil {
		panic(1)
	}

	tty.AlternateScreenBuffer()
	tty.EnableANSIMode()
	tty.MoveCurPos(tty.ws.Row, 0)
}

func (tty *Tty) ShutdownTtyRoutine(c chan os.Signal) {
	signal.Notify(c, unix.SIGHUP, unix.SIGINT, unix.SIGTERM, unix.SIGQUIT)

	go func() {
		<-c

		fmt.Print(DALT_SCREEN_BUFF)
		tty.EnableEcho()
		tty.EnableCursor()
		tty.DisableANSIMode()
		unix.Exit(0)
	}()
}

func (tty *Tty) EnableEcho() {
	tty.stdinTermios.Lflag |= unix.ECHO
	unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)
}

func (tty *Tty) DisableEcho() {
	tty.stdinTermios.Lflag &^= unix.ECHO
	unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)
}

func (tty *Tty) EnableICANON() {
	tty.stdinTermios.Lflag |= unix.ICANON
	unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)
}

func (tty *Tty) DisableICANON() {
	tty.stdinTermios.Lflag &^= unix.ICANON
	unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)
}

func (tty *Tty) EnableISIG() {
	tty.stdinTermios.Lflag |= unix.ISIG
	unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)
}

func (tty *Tty) DisableISIG() {
	tty.stdinTermios.Lflag &^= unix.ISIG
	unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)
}

func (tty *Tty) EnableANSIMode() {
	fmt.Printf("\x1b[?2h")
}

func (tty *Tty) DisableANSIMode() {
	fmt.Printf("\x1b[?2l")
}

func (tty *Tty) Rows() uint16 {
	return tty.ws.Row
}

func (tty *Tty) Columns() uint16 {
	return tty.ws.Col
}

func (tty *Tty) EnableBlinkingCursor() {
	fmt.Printf(ENABLE_BLINKING_CURSOR)
}

func (tty *Tty) DisableBlinkingCursor() {
	fmt.Printf(DISABLE_BLINKING_CURSOR)
}

func (tty *Tty) EnableCursor() {
	fmt.Printf(ENABLE_SHOW_CURSOR)
}

func (tty *Tty) DisableCursor() {
	fmt.Printf(DISABLE_SHOW_CURSOR)
}
