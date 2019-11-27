package logger

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/tj/go-spin"
)

type Logger struct {
	spinnerStopCh chan bool
	spinnerMsg    string
	spinnerArgs   []interface{}
	isSilent      bool
	isVerbose     bool
}

func NewLogger() *Logger {
	return &Logger{}
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

	fmt.Println("")
}

func (l *Logger) Finish() {
	if l == nil || l.isSilent {
		return
	}

	fmt.Println("")
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Printf("    ")
	fmt.Println(fmt.Sprintf(msg, args...))
	fmt.Println("")
}

func (l *Logger) Info(msg string, args ...interface{}) {
	if l == nil || l.isSilent || !l.isVerbose {
		return
	}

	fmt.Printf("    ")
	fmt.Println(fmt.Sprintf(msg, args...))
	fmt.Println("")
}

func (l *Logger) ActionWithoutSpinner(msg string, args ...interface{}) {
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

func (l *Logger) ChildActionWithoutSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	fmt.Printf("    • ")
	fmt.Println(fmt.Sprintf(msg, args...))
}

func (l *Logger) ActionWithSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	s := spin.New()

	fmt.Printf("  • ")
	fmt.Printf(msg, args...)
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

func (l *Logger) ChildActionWithSpinner(msg string, args ...interface{}) {
	if l == nil || l.isSilent {
		return
	}

	s := spin.New()

	fmt.Printf("    • ")
	fmt.Printf(msg, args...)
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

func (l *Logger) FinishChildSpinner() {
	if l == nil || l.isSilent {
		return
	}

	green := color.New(color.FgHiGreen)

	fmt.Printf("\r")
	fmt.Printf("    • ")
	fmt.Printf(l.spinnerMsg, l.spinnerArgs...)
	green.Printf(" ✓")
	fmt.Printf("  \n")

	l.spinnerStopCh <- true
	close(l.spinnerStopCh)
}

func (l *Logger) FinishSpinner() {
	if l == nil || l.isSilent {
		return
	}

	green := color.New(color.FgHiGreen)

	fmt.Printf("\r")
	fmt.Printf("  • ")
	fmt.Printf(l.spinnerMsg, l.spinnerArgs...)
	green.Printf(" ✓")
	fmt.Printf("  \n")

	l.spinnerStopCh <- true
	close(l.spinnerStopCh)
}

func (l *Logger) FinishSpinnerWithError() {
	if l == nil || l.isSilent {
		return
	}

	red := color.New(color.FgHiRed)

	fmt.Printf("\r")
	fmt.Printf("  • ")
	fmt.Printf(l.spinnerMsg, l.spinnerArgs...)
	red.Printf(" ✗")
	fmt.Printf("  \n")

	l.spinnerStopCh <- true
	close(l.spinnerStopCh)
}

func (l *Logger) Error(err error) {
	if l == nil || l.isSilent {
		return
	}

	c := color.New(color.FgHiRed)
	c.Printf("  • ")
	c.Println(fmt.Sprintf("%#v", err))
}
