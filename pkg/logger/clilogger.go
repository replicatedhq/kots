package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/tj/go-spin"
)

type CLILogger struct {
	spinnerStopCh chan bool
	spinnerMsg    string
	spinnerArgs   []interface{}
	isSilent      bool
	isVerbose     bool
}

func NewCLILogger() *CLILogger {
	return &CLILogger{}
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

	fmt.Println("")
}

func (l *CLILogger) Finish() {
	if l == nil || l.isSilent {
		return
	}

	fmt.Println("")
}

func (l *CLILogger) Debug(msg string, args ...interface{}) {
	if l == nil || l.isSilent || !l.isVerbose {
		return
	}

	fmt.Printf("    ")
	fmt.Println(fmt.Sprintf(msg, args...))
	fmt.Println("")
}

func (l *CLILogger) Info(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Printf("    ")
	fmt.Println(fmt.Sprintf(msg, args...))
	fmt.Println("")
}

func (l *CLILogger) ActionWithoutSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	if msg == "" {
		fmt.Println("")
		return
	}

	fmt.Printf("  • ")
	fmt.Println(fmt.Sprintf(msg, args...))
}

func (l *CLILogger) ChildActionWithoutSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Printf("    • ")
	fmt.Println(fmt.Sprintf(msg, args...))
}

func (l *CLILogger) ActionWithSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Printf("  • ")
	fmt.Printf(msg, args...)

	if isatty.IsTerminal(os.Stdout.Fd()) {
		s := spin.New()

		fmt.Printf(" %s", s.Next())

		l.spinnerStopCh = make(chan bool)
		l.spinnerMsg = msg
		l.spinnerArgs = args

		go func() {
			for {
				select {
				case <-l.spinnerStopCh:
					return
				case <-time.After(time.Millisecond * 100):
					fmt.Printf("\r")
					fmt.Printf("  • ")
					fmt.Printf(msg, args...)
					fmt.Printf(" %s", s.Next())
				}
			}
		}()
	}
}

func (l *CLILogger) ChildActionWithSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Printf("    • ")
	fmt.Printf(msg, args...)

	if isatty.IsTerminal(os.Stdout.Fd()) {
		s := spin.New()

		fmt.Printf(" %s", s.Next())

		l.spinnerStopCh = make(chan bool)
		l.spinnerMsg = msg
		l.spinnerArgs = args

		go func() {
			for {
				select {
				case <-l.spinnerStopCh:
					return
				case <-time.After(time.Millisecond * 100):
					fmt.Printf("\r")
					fmt.Printf("    • ")
					fmt.Printf(msg, args...)
					fmt.Printf(" %s", s.Next())
				}
			}
		}()
	}
}

func (l *CLILogger) FinishChildSpinner() {
	if l == nil || l.isSilent {
		return
	}

	green := color.New(color.FgHiGreen)

	fmt.Printf("\r")
	fmt.Printf("    • ")
	fmt.Printf(l.spinnerMsg, l.spinnerArgs...)
	green.Printf(" ✓")
	fmt.Printf("  \n")

	if isatty.IsTerminal(os.Stdout.Fd()) {
		l.spinnerStopCh <- true
		close(l.spinnerStopCh)
	}
}

func (l *CLILogger) FinishSpinner() {
	if l == nil || l.isSilent {
		return
	}

	green := color.New(color.FgHiGreen)

	fmt.Printf("\r")
	fmt.Printf("  • ")
	fmt.Printf(l.spinnerMsg, l.spinnerArgs...)
	green.Printf(" ✓")
	fmt.Printf("  \n")

	if isatty.IsTerminal(os.Stdout.Fd()) {
		l.spinnerStopCh <- true
		close(l.spinnerStopCh)
	}
}

func (l *CLILogger) FinishSpinnerWithError() {
	if l == nil || l.isSilent {
		return
	}

	red := color.New(color.FgHiRed)

	fmt.Printf("\r")
	fmt.Printf("  • ")
	fmt.Printf(l.spinnerMsg, l.spinnerArgs...)
	red.Printf(" ✗")
	fmt.Printf("  \n")

	if isatty.IsTerminal(os.Stdout.Fd()) {
		l.spinnerStopCh <- true
		close(l.spinnerStopCh)
	}
}

// FinishSpinnerWithWarning if no color is provided, color.FgYellow will be used
func (l *CLILogger) FinishSpinnerWithWarning(c *color.Color) {
	if l == nil || l.isSilent {
		return
	}

	if c == nil {
		c = color.New(color.FgYellow)
	}

	fmt.Printf("\r")
	fmt.Printf("  • ")
	fmt.Printf(l.spinnerMsg, l.spinnerArgs...)
	c.Printf(" !")
	fmt.Printf("  \n")

	if isatty.IsTerminal(os.Stdout.Fd()) {
		l.spinnerStopCh <- true
		close(l.spinnerStopCh)
	}
}

func (l *CLILogger) Error(err error) {
	if l == nil || l.isSilent {
		return
	}

	c := color.New(color.FgHiRed)
	c.Printf("  • ")
	c.Println(fmt.Sprintf("%#v", err))
}

func (l *CLILogger) Errorf(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	c := color.New(color.FgHiRed)
	c.Printf("  • ")
	c.Println(fmt.Sprintf(msg, args...))
}
