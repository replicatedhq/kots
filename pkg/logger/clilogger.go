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

type CLILogger struct {
	writer           io.Writer
	spinnerStopCh    chan bool
	spinnerMsg       string
	spinnerArgs      []interface{}
	isSpinnerRunning bool
	isSilent         bool
	isVerbose        bool
}

func NewCLILogger(w io.Writer) *CLILogger {
	return &CLILogger{writer: w}
}

func (l *CLILogger) Silence() {
	if l == nil {
		return
	}
	l.isSilent = true
}

func (l *CLILogger) Verbose() {
	if l == nil {
		return
	}
	l.isVerbose = true
}

func (l *CLILogger) Initialize() {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintln(l.writer, "")
}

func (l *CLILogger) Finish() {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintln(l.writer, "")
}

func (l *CLILogger) Debug(msg string, args ...interface{}) {
	if l == nil || l.isSilent || !l.isVerbose {
		return
	}

	fmt.Fprintf(l.writer, "    ")
	fmt.Fprintln(l.writer, fmt.Sprintf(msg, args...))
	fmt.Fprintln(l.writer, "")
}

func (l *CLILogger) Info(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintf(l.writer, "    ")
	fmt.Fprintln(l.writer, fmt.Sprintf(msg, args...))
	fmt.Fprintln(l.writer, "")
}

func (l *CLILogger) ActionWithoutSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	if msg == "" {
		fmt.Fprintln(l.writer, "")
		return
	}

	fmt.Fprintf(l.writer, "  • ")
	fmt.Fprintln(l.writer, fmt.Sprintf(msg, args...))
}

func (l *CLILogger) ActionWithoutSpinnerWarning(msg string, c *color.Color, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	if msg == "" {
		fmt.Fprintln(l.writer, "")
		return
	}

	if c == nil {
		c = color.New(color.FgYellow)
	}

	fmt.Fprintf(l.writer, "  • ")
	fmt.Fprintf(l.writer, msg, args...)
	c.Fprintf(l.writer, " !")
	fmt.Fprintf(l.writer, "  \n")
}

func (l *CLILogger) ChildActionWithoutSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintf(l.writer, "    • ")
	fmt.Fprintln(l.writer, fmt.Sprintf(msg, args...))
}

func (l *CLILogger) ActionWithSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintf(l.writer, "  • ")
	fmt.Fprintf(l.writer, msg, args...)

	if !l.IsTerminal() {
		fmt.Fprintln(l.writer)
		return
	}

	s := spin.New()

	fmt.Fprintf(l.writer, " %s", s.Next())

	l.spinnerStopCh = make(chan bool)
	l.spinnerMsg = msg
	l.spinnerArgs = args
	l.isSpinnerRunning = true

	go func() {
		for {
			select {
			case <-l.spinnerStopCh:
				return
			case <-time.After(time.Millisecond * 100):
				fmt.Fprintf(l.writer, "\r")
				fmt.Fprintf(l.writer, "  • ")
				fmt.Fprintf(l.writer, msg, args...)
				fmt.Fprintf(l.writer, " %s", s.Next())
			}
		}
	}()
}

func (l *CLILogger) ChildActionWithSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Fprintf(l.writer, "    • ")
	fmt.Fprintf(l.writer, msg, args...)

	if !l.IsTerminal() {
		fmt.Fprintln(l.writer)
		return
	}

	s := spin.New()

	fmt.Fprintf(l.writer, " %s", s.Next())

	l.spinnerStopCh = make(chan bool)
	l.spinnerMsg = msg
	l.spinnerArgs = args
	l.isSpinnerRunning = true

	go func() {
		for {
			select {
			case <-l.spinnerStopCh:
				return
			case <-time.After(time.Millisecond * 100):
				fmt.Fprintf(l.writer, "\r")
				fmt.Fprintf(l.writer, "    • ")
				fmt.Fprintf(l.writer, msg, args...)
				fmt.Fprintf(l.writer, " %s", s.Next())
			}
		}
	}()
}

func (l *CLILogger) FinishChildSpinner() {
	if l == nil || l.isSilent || !l.isSpinnerRunning {
		return
	}

	if !l.IsTerminal() {
		fmt.Fprintln(l.writer, "    •  ✓")
		return
	}

	green := color.New(color.FgHiGreen)

	fmt.Fprintf(l.writer, "\r")
	fmt.Fprintf(l.writer, "    • ")
	fmt.Fprintf(l.writer, l.spinnerMsg, l.spinnerArgs...)
	green.Fprintf(l.writer, " ✓")
	fmt.Fprintf(l.writer, "  \n")

	l.spinnerStopCh <- true
	l.isSpinnerRunning = false
	close(l.spinnerStopCh)
}

func (l *CLILogger) FinishSpinner() {
	if l == nil || l.isSilent || !l.isSpinnerRunning {
		return
	}

	if !l.IsTerminal() {
		fmt.Fprintln(l.writer, "  •  ✓")
		return
	}

	green := color.New(color.FgHiGreen)

	fmt.Fprintf(l.writer, "\r")
	fmt.Fprintf(l.writer, "  • ")
	fmt.Fprintf(l.writer, l.spinnerMsg, l.spinnerArgs...)
	green.Fprintf(l.writer, " ✓")
	fmt.Fprintf(l.writer, "  \n")

	l.spinnerStopCh <- true
	l.isSpinnerRunning = false
	close(l.spinnerStopCh)
}

func (l *CLILogger) FinishSpinnerWithError() {
	if l == nil || l.isSilent || !l.isSpinnerRunning {
		return
	}

	if !l.IsTerminal() {
		fmt.Fprintln(l.writer, "  •  ✗")
		return
	}

	red := color.New(color.FgHiRed)

	fmt.Fprintf(l.writer, "\r")
	fmt.Fprintf(l.writer, "  • ")
	fmt.Fprintf(l.writer, l.spinnerMsg, l.spinnerArgs...)
	red.Fprintf(l.writer, " ✗")
	fmt.Fprintf(l.writer, "  \n")

	l.spinnerStopCh <- true
	l.isSpinnerRunning = false
	close(l.spinnerStopCh)
}

// FinishSpinnerWithWarning if no color is provided, color.FgYellow will be used
func (l *CLILogger) FinishSpinnerWithWarning(c *color.Color) {
	if l == nil || l.isSilent || !l.isSpinnerRunning {
		return
	}

	if !l.IsTerminal() {
		fmt.Fprintln(l.writer, "  •  !")
		return
	}

	if c == nil {
		c = color.New(color.FgYellow)
	}

	fmt.Fprintf(l.writer, "\r")
	fmt.Fprintf(l.writer, "  • ")
	fmt.Fprintf(l.writer, l.spinnerMsg, l.spinnerArgs...)
	c.Fprintf(l.writer, " !")
	fmt.Fprintf(l.writer, "  \n")

	l.spinnerStopCh <- true
	l.isSpinnerRunning = false
	close(l.spinnerStopCh)
}

func (l *CLILogger) Error(err error) {
	if l == nil || l.isSilent {
		return
	}

	c := color.New(color.FgHiRed)
	c.Fprintf(l.writer, "  • ")
	c.Fprintln(l.writer, fmt.Sprintf("%#v", err))
}

func (l *CLILogger) Errorf(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	c := color.New(color.FgHiRed)
	c.Fprintf(l.writer, "  • ")
	c.Fprintln(l.writer, fmt.Sprintf(msg, args...))
}

func (l *CLILogger) IsTerminal() bool {
	file, ok := l.writer.(*os.File)
	if ok {
		return isatty.IsTerminal(file.Fd())
	}
	return false
}
