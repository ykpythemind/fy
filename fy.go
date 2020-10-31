package fy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type Cli struct {
	In         io.ReadSeeker
	Stdout     io.Writer
	keyCh      chan rune
	quitCh     chan struct{}
	filterCh   chan *FilterResult
	filter     Filter
	filtered   *FilterResult
	inputRunes []rune
	Screen     tcell.Screen
}

func New() (*Cli, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	// wip---
	by, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	if len(by) == 0 {
		return nil, errors.New("no input") // wip
	}
	reader := bytes.NewReader(by)
	// ---

	if err := screen.Init(); err != nil {
		return nil, err
	}

	filter := &FilterImpl{}

	app := &Cli{
		In: reader, Stdout: os.Stdout, // wip
		keyCh: make(chan rune), Screen: screen,
		quitCh:   make(chan struct{}),
		filterCh: make(chan *FilterResult),
		filter:   filter,
		filtered: &FilterResult{},
	}
	return app, nil
}

func (app *Cli) Run() error {
	go app.handleEvent()
	go app.handleKeyInput()
	go app.doFilter()

	app.render()

	<-app.quitCh

	app.exit()

	fmt.Printf("debug: %s\n", string(app.inputRunes))
	fmt.Printf("filtered: %v\n", app.filtered)

	return nil
}

func (app *Cli) handleKeyInput() {
	for {
		r := <-app.keyCh

		app.inputRunes = append(app.inputRunes, r)
		go app.doFilter()
		app.render()
	}
}

func (app *Cli) doFilter() {
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

func (app *Cli) render() {
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

func (app *Cli) handleEvent() {
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

func (app *Cli) parseKey(b []byte) (rune, int) {
	return utf8.DecodeRune(b)
}

func (app *Cli) exit() {
	app.Screen.Fini()
}
