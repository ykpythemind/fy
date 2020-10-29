package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type App struct {
	In         io.ReadSeeker
	Stdout     io.Writer
	keyCh      chan rune
	quitCh     chan struct{}
	filterCh   chan FilterResult
	filter     Filter
	filtered   FilterResult
	inputRunes []rune
	Screen     tcell.Screen
}

func New() (*App, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	if err := screen.Init(); err != nil {
		return nil, err
	}

	filter := &FilterImpl{}

	// wip---
	by, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(by)
	// ---

	app := &App{
		In: reader, Stdout: os.Stdout, // wip
		keyCh: make(chan rune), Screen: screen,
		quitCh:   make(chan struct{}),
		filterCh: make(chan FilterResult),
		filter:   filter,
	}
	return app, nil
}

func (app *App) Run() error {
	go app.handleEvent()
	go app.handleKeyInput()
	go app.doFilter()

	app.render()

	<-app.quitCh

	app.exit()

	fmt.Printf("debug: %s\n", string(app.inputRunes))

	return nil
}

func (app *App) handleKeyInput() {
	for {
		r := <-app.keyCh

		app.inputRunes = append(app.inputRunes, r)
		go app.doFilter()
		app.render()
	}
}

func (app *App) doFilter() {
	// 前に実行してたやつをキャンセルしたほうがええかも

	context := context.Background()
	err := app.filter.Run(context, app.In, app.filterCh)
	if err != nil {
		// todo do some handling
		return
	}

	result := <-app.filterCh
	app.filtered = result
	app.render()
}

func (app *App) render() {
	query := []rune("[QUERY]")
	queryLen := len(query)

	for i, r := range []rune(query) {
		app.Screen.SetContent(i, 0, r, nil, tcell.StyleDefault)
	}
	for i, r := range app.inputRunes {
		app.Screen.SetContent(i+queryLen+1, 0, r, nil, tcell.StyleDefault)
	}

	for i, line := range app.filtered.Matched {
		for x, r := range []rune(line) {
			app.Screen.SetContent(x, i+1, r, nil, tcell.StyleDefault)
		}
	}

	app.Screen.Show()
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
