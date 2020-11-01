package fy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/gdamore/tcell/v2"
)

const queryMarker = "[QUERY]"

type CLI struct {
	In         io.ReadSeeker
	Stdout     io.Writer
	keyCh      chan rune
	quitCh     chan struct{}
	filterCh   chan filterResult
	filter     filter
	filtered   filterResult
	inputRunes []rune
	Screen     tcell.Screen
	mu         sync.Mutex
}

func New() (*CLI, error) {
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

	filter := &cliFilter{}

	cli := &CLI{
		In:     reader,
		Stdout: os.Stdout, // wip
		keyCh:  make(chan rune), Screen: screen,
		quitCh:   make(chan struct{}),
		filterCh: make(chan filterResult, 1),
		filter:   filter,
		filtered: filterResult{},
	}
	return cli, nil
}

func (cli *CLI) Run() error {
	go cli.handleEvent()
	go cli.handleKeyInput()
	go cli.doFilter()

	cli.render()

	<-cli.quitCh

	cli.exit()

	fmt.Printf("debug: %s\n", string(cli.inputRunes))
	fmt.Printf("filtered: %v\n", cli.filtered)

	return nil
}

func (cli *CLI) handleKeyInput() {
	for {
		r := <-cli.keyCh

		cli.mu.Lock()
		cli.inputRunes = append(cli.inputRunes, r)
		cli.mu.Unlock()

		go cli.doFilter()
		cli.render()
	}
}

func (cli *CLI) doFilter() {
	// 前に実行してたやつをキャンセルしたほうがええかも
	context := context.Background()

	err := cli.filter.Run(context, cli.In, cli.filterCh)
	if err != nil {
		// todo do some handling
		return
	}

	result := <-cli.filterCh

	cli.mu.Lock()
	cli.filtered = result
	cli.mu.Unlock()
}

func (cli *CLI) render() {
	query := []rune(queryMarker)
	queryLen := len(query)

	for i, r := range []rune(query) {
		cli.Screen.SetContent(i, 0, r, nil, tcell.StyleDefault)
	}
	for i, r := range cli.inputRunes {
		cli.Screen.SetContent(i+queryLen+1, 0, r, nil, tcell.StyleDefault)
	}

	_, y := cli.Screen.Size()
	matchedLinesHeight := y - 1

	for i, line := range cli.filtered.matched {
		if i > matchedLinesHeight {
			break
		}
		for x, r := range []rune(line) {
			cli.Screen.SetContent(x, i+1, r, nil, tcell.StyleDefault)
		}
	}

	cli.Screen.Show()
}

func (cli *CLI) handleEvent() {
	for {
		ev := cli.Screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventKey:
			key := ev.Key()
			if key == tcell.KeyEscape {
				close(cli.quitCh)
				return
			}
			if _, isSpecialKey := tcell.KeyNames[key]; isSpecialKey {
				// ignore
				continue
			}
			r := ev.Rune()
			cli.keyCh <- r
		case *tcell.EventResize:
			cli.Screen.Sync()
		}
	}
}

func (cli *CLI) exit() {
	cli.Screen.Fini()
}
