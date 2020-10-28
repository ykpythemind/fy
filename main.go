package main

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"unicode/utf8"
)

const (
	ControlA = 1
)

type App struct {
	Stdin  io.Reader
	Stdout io.Writer
	keyCh  chan rune
	strs   []rune
}

func New() *App {
	app := &App{Stdin: os.Stdin, Stdout: os.Stdout, keyCh: make(chan rune)}
	return app
}

func (app *App) Run() error {
	go app.readKeys()

	app.handleKeyInput()

	return nil
}

func (app *App) handleKeyInput() {
	for {
		r := <-app.keyCh

		switch r {
		case ControlA:
			// exit
			str := string(app.strs)
			_, _ = app.Stdout.Write([]byte(str))
			return
		default:
			fmt.Println(r)
			fmt.Println("aaaaa")
			app.strs = append(app.strs, r)
		}
	}
}

func (app *App) readKeys() {
	buf := make([]byte, 64)

	for {
		if n, err := syscall.Read(0, buf); err == nil {
			b := buf[:n]
			for {
				r, n := app.parseKey(b)

				if n == 0 {
					break
				}

				app.keyCh <- r
				b = buf[n:]
			}
		}
	}
}

func (app *App) parseKey(b []byte) (rune, int) {
	return utf8.DecodeRune(b)
}

func main() {
	app := New()

	err := app.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
