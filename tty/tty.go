package tty

import (
	"golang.org/x/sys/unix"
	"fmt"
	"os"
	"os/signal"
	"io"
	"slices"
	"bufio"
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
	SCROLL_REGION = "\x1b[%d;%dr"
	ERASE_ENTIRE_SCREEN = "\x1b[2J"
	ERASE_ENTIRE_LINE = "\x1b[2K"
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
	tty.stdinTermios.Iflag |= unix.IUTF8
	tty.stdinTermios.Lflag |= unix.IEXTEN
	tty.stdinTermios.Lflag &^= unix.ICANON | unix.ECHO | unix.ECHOCTL
	err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)

	if err != nil {
		panic(1)
	}

	unix.Write(unix.Stdout, []byte("\x1b[?1h"))
	tty.ClearScreen()
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
		unix.Write(unix.Stdout, []byte("\x1b[?1h"))
		unix.Write(unix.Stdout, []byte("\x1b[?67h"))
		tty.EraseEntireLine()
		tty.MoveCurPos(tty.ws.Row, 0)
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

func (tty *Tty) SetScrollRegion(top, bottom uint16) {
	fmt.Printf(SCROLL_REGION, top, bottom)
}

func (tty *Tty) ClearScreen() {
	fmt.Printf(ERASE_ENTIRE_SCREEN)
}

func (tty *Tty) EraseEntireLine() {
	fmt.Print(ERASE_ENTIRE_LINE)
}

func (tty *Tty) ReadLine() (string, error) {
	reader := bufio.NewReader(os.Stdin) // Since os.Stdin doesn't implement ReadByte, need to use bufio.NewReader
	lineBuf := []rune{} // Buffer the content typed by the user
	curPos := 0 // Track the position of the cursor in the line

	for {
		c, err := reader.ReadByte()

		if err != nil {
			if err == io.EOF {
				break
			}
		}

		if c == '\x1b' {
			cmd_seq, err := reader.Peek(2) // Peeks into the next 2 bytes and return them
			if err != nil {
				panic(1)
			}
			if cmd_seq[0] == '[' || cmd_seq[0] == 'O' {
				if cmd_seq[1] == 'D' && curPos > 0 {
					os.Stdout.WriteString("\x1b[D")
					curPos--
				} else if cmd_seq[1] == 'C' && curPos < len(lineBuf) {
					os.Stdout.WriteString("\x1b[C")
					curPos++
				} else if cmd_seq[1] == '\x33' {
					if curPos < len(lineBuf) {
						lineBuf = append(lineBuf[:curPos], lineBuf[curPos+1:]...)
						str := fmt.Sprintf("\x1b[K%s\x1b[%dG", string(lineBuf[curPos:]), curPos+3)
						os.Stdout.WriteString(str)
					}
				}
			}
			reader.Discard(reader.Buffered())
			continue
		}

		if c == '\x7F' { // It's the BACKSPACE char
			if curPos > 0 {
				lineBuf = append(lineBuf[:curPos-1], lineBuf[curPos:]...)
				curPos--
				os.Stdout.WriteString("\b\x1b[1P")
			}
			continue
		}

		if c == '\r' || c == '\n' {
			os.Stdout.WriteString(string(c))
			return string(lineBuf), nil
		}

		if curPos < len(lineBuf) {
			lineBuf = slices.Insert(lineBuf, curPos, rune(c))
			os.Stdout.WriteString("\x1b[4h")
		} else {
			lineBuf = append(lineBuf, rune(c))
		}
		os.Stdout.WriteString(string(c))
		curPos++
	}
	return "", fmt.Errorf("ERROR")
}

func (tty *Tty) SttySane() {
	tty.stdinTermios.Lflag &^= unix.ICANON | unix.ECHO | unix.ECHOCTL
	err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)

	if err != nil {
		panic(1)
	}
}

func (tty *Tty) GetTermios() unix.Termios {
	return *tty.stdinTermios
}

func (tty *Tty) SetTermios(ntermios *unix.Termios) error {
	err := unix.IoctlSetTermios(unix.Stdin, unix.TCSETS, tty.stdinTermios)

	if err != nil {
		return err
	}

	return nil
}
