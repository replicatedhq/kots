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
}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Initialize() {
	fmt.Println("")
}

func (l *Logger) Finish() {
	fmt.Println("")
}

func (l *Logger) ActionWithoutSpinner(msg string, args ...interface{}) {
	white := color.New(color.FgHiWhite)
	white.Printf("  • ")
	white.Println(fmt.Sprintf(msg, args...))
}

func (l *Logger) ActionWithSpinner(msg string, args ...interface{}) {
	s := spin.New()

	c := color.New(color.FgHiCyan)
	c.Printf("  • ")
	c.Printf(msg, args...)
	c.Printf(" %s", s.Next())

	l.spinnerStopCh = make(chan bool)
	l.spinnerMsg = msg
	l.spinnerArgs = args

	go func() {
		for {
			select {
			case <-l.spinnerStopCh:
				return
			case <-time.After(time.Millisecond * 100):
				c.Printf("\r")
				c.Printf("  • ")
				c.Printf(msg, args...)
				c.Printf(" %s", s.Next())
			}
		}
	}()
}

func (l *Logger) FinishSpinner() {
	white := color.New(color.FgHiWhite)
	green := color.New(color.FgHiGreen)

	white.Printf("\r")
	white.Printf("  • ")
	white.Printf(l.spinnerMsg, l.spinnerArgs...)
	green.Printf(" ✓")
	white.Printf("  \n")

	l.spinnerStopCh <- true
	close(l.spinnerStopCh)
}

func (l *Logger) Error(err error) {
	c := color.New(color.FgHiRed)
	c.Printf("  • ")
	c.Println(fmt.Sprintf("%#v", err))
}
