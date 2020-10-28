package main

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type App struct {
	Stdin  io.Reader
	Stdout io.Writer
	keyCh  chan rune
	quitCh chan struct{}
	runes  []rune
	Screen tcell.Screen
}

func New() (*App, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	if err := screen.Init(); err != nil {
		return nil, err
	}

	app := &App{
		Stdin: os.Stdin, Stdout: os.Stdout, keyCh: make(chan rune), Screen: screen,
		quitCh: make(chan struct{}),
	}
	return app, nil
}

func (app *App) Run() error {
	go app.handleEvent()
	go app.handleKeyInput()

	<-app.quitCh

	app.exit()

	fmt.Printf("debug: %s\n", string(app.runes))

	return nil
}

func (app *App) handleKeyInput() {
	for {
		r := <-app.keyCh

		app.runes = append(app.runes, r)
		for i, r := range app.runes {
			app.Screen.SetContent(i, 0, r, nil, tcell.StyleDefault)
		}
		app.Screen.Show()
	}
}

func (app *App) handleEvent() {
	for {
		ev := app.Screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventKey:
			key := ev.Key()
			if key == tcell.KeyEscape {
				close(app.quitCh)
				return
			}
			if _, isSpecialKey := tcell.KeyNames[key]; isSpecialKey {
				// ignore
				continue
			}
			r := ev.Rune()
			app.keyCh <- r
		case *tcell.EventResize:
			app.Screen.Sync()
		}
	}
}

func (app *App) parseKey(b []byte) (rune, int) {
	return utf8.DecodeRune(b)
}

func (app *App) exit() {
	app.Screen.Fini()
}

func main() {
	app, err := New()
	if err != nil {
		fmt.Fprint(os.Stdout, err)
		os.Exit(1)
	}

	err = app.Run()
	if err != nil {
		fmt.Fprint(os.Stdout, err)
		os.Exit(1)
	}
}
