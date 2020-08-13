package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/tj/go-spin"
)

type Logger struct {
	w io.Writer
	spinnerStopCh chan bool
	spinnerMsg    string
	spinnerArgs   []interface{}
	isSilent      bool
	isVerbose     bool
}

func NewLogger(writer io.Writer) *Logger {
	return &Logger{
		w: writer,
	}
}

func (l *Logger) Silence() {
	if l == nil {
		return
	}
	l.isSilent = true
}

func (l *Logger) Verbose() {
	if l == nil {
		return
	}
	l.isVerbose = true
}

func (l *Logger) Initialize() {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintln(l.w, "")
}

func (l *Logger) Finish() {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintln(l.w, "")
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	if l == nil || l.isSilent || !l.isVerbose {
		return
	}

	fmt.Fprintf(l.w, "    ")
	fmt.Fprintln(l.w, fmt.Sprintf(msg, args...))
	fmt.Fprintln(l.w, "")
}

func (l *Logger) Info(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintf(l.w, "    ")
	fmt.Fprintln(l.w, fmt.Sprintf(msg, args...))
	fmt.Fprintln(l.w, "")
}

func (l *Logger) ActionWithoutSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	if msg == "" {
		fmt.Fprintln(l.w, "")
		return
	}

	fmt.Fprintf(l.w, "  • ")
	fmt.Fprintln(l.w, fmt.Sprintf(msg, args...))
}

func (l *Logger) ChildActionWithoutSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintf(l.w, "    • ")
	fmt.Fprintln(l.w, fmt.Sprintf(msg, args...))
}

func (l *Logger) ActionWithSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintf(l.w, "  • ")
	fmt.Fprintf(l.w, msg, args...)

	if isatty.IsTerminal(os.Stdout.Fd()) {
		s := spin.New()

		fmt.Fprintf(l.w, " %s", s.Next())

		l.spinnerStopCh = make(chan bool)
		l.spinnerMsg = msg
		l.spinnerArgs = args

		go func() {
			for {
				select {
				case <-l.spinnerStopCh:
					return
				case <-time.After(time.Millisecond * 100):
					fmt.Fprintf(l.w, "\r")
					fmt.Fprintf(l.w, "  • ")
					fmt.Fprintf(l.w, msg, args...)
					fmt.Fprintf(l.w, " %s", s.Next())
				}
			}
		}()
	}
}

func (l *Logger) ChildActionWithSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintf(l.w, "    • ")
	fmt.Fprintf(l.w, msg, args...)

	if isatty.IsTerminal(os.Stdout.Fd()) {
		s := spin.New()

		fmt.Fprintf(l.w, " %s", s.Next())

		l.spinnerStopCh = make(chan bool)
		l.spinnerMsg = msg
		l.spinnerArgs = args

		go func() {
			for {
				select {
				case <-l.spinnerStopCh:
					return
				case <-time.After(time.Millisecond * 100):
					fmt.Fprintf(l.w, "\r")
					fmt.Fprintf(l.w, "    • ")
					fmt.Fprintf(l.w, msg, args...)
					fmt.Fprintf(l.w, " %s", s.Next())
				}
			}
		}()
	}
}

func (l *Logger) FinishChildSpinner() {
	if l == nil || l.isSilent {
		return
	}

	green := color.New(color.FgHiGreen)

	fmt.Fprintf(l.w, "\r")
	fmt.Fprintf(l.w, "    • ")
	fmt.Fprintf(l.w, l.spinnerMsg, l.spinnerArgs...)
	green.Fprintf(l.w, " ✓")
	fmt.Fprintf(l.w, "  \n")

	if isatty.IsTerminal(os.Stdout.Fd()) {
		l.spinnerStopCh <- true
		close(l.spinnerStopCh)
	}
}

func (l *Logger) FinishSpinner() {
	if l == nil || l.isSilent {
		return
	}

	green := color.New(color.FgHiGreen)

	fmt.Fprintf(l.w, "\r")
	fmt.Fprintf(l.w, "  • ")
	fmt.Fprintf(l.w, l.spinnerMsg, l.spinnerArgs...)
	green.Fprintf(l.w, " ✓")
	fmt.Fprintf(l.w, "  \n")

	if isatty.IsTerminal(os.Stdout.Fd()) {
		l.spinnerStopCh <- true
		close(l.spinnerStopCh)
	}
}

func (l *Logger) FinishSpinnerWithError() {
	if l == nil || l.isSilent {
		return
	}

	red := color.New(color.FgHiRed)

	fmt.Fprintf(l.w, "\r")
	fmt.Fprintf(l.w, "  • ")
	fmt.Fprintf(l.w, l.spinnerMsg, l.spinnerArgs...)
	red.Fprintf(l.w, " ✗")
	fmt.Fprintf(l.w, "  \n")

	if isatty.IsTerminal(os.Stdout.Fd()) {
		l.spinnerStopCh <- true
		close(l.spinnerStopCh)
	}
}

func (l *Logger) Error(err error) {
	if l == nil || l.isSilent {
		return
	}

	c := color.New(color.FgHiRed)
	c.Fprintf(l.w, "  • ")
	c.Fprintln(l.w, fmt.Sprintf("%#v", err))
}
