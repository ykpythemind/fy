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
	input      io.ReadSeeker
	output     io.Writer
	keyCh      chan rune
	quitCh     chan struct{}
	filterCh   chan []matched
	filter     filter
	selectCh   chan struct{}
	current    matched
	matched    []matched
	inputRunes []rune
	Screen     tcell.Screen
	mu         sync.Mutex
	debug      bool
}

func New(debug bool) (*CLI, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	// todo:
	by, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	if len(by) == 0 {
		// pecoは以下のエラー出す
		//  Error: failed to setup input source: you must supply something to work with via filename or stdin
		return nil, errors.New("no input") // wip
	}
	reader := bytes.NewReader(by)
	// ---

	if err := screen.Init(); err != nil {
		return nil, err
	}

	filter := &cliFilter{}

	cli := &CLI{
		input:    reader,
		output:   os.Stdout, // wip
		keyCh:    make(chan rune),
		Screen:   screen,
		quitCh:   make(chan struct{}),
		filterCh: make(chan []matched, 1),
		selectCh: make(chan struct{}),
		current:  matched{},
		filter:   filter,
		matched:  []matched{},
		debug:    debug,
	}
	return cli, nil
}

func (cli *CLI) Run() error {
	go cli.handleEvent()
	go cli.handleKeyInput()
	go cli.doFilter()

	cli.render()

	selected := false

	select {
	case <-cli.quitCh:
	case <-cli.selectCh:
		selected = true
	}

	cli.exit()

	if selected {
		fmt.Fprintf(cli.output, "%s", cli.current.line)
	}

	if cli.debug {
		fmt.Printf("debug: %s\n", string(cli.inputRunes))
		fmt.Printf("matched: %v\n", cli.matched)
	}

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

	err := cli.filter.Run(context, string(cli.inputRunes), cli.input, cli.filterCh)
	if err != nil {
		// todo do some handling
		return
	}

	result := <-cli.filterCh

	cli.mu.Lock()
	cli.matched = result
	if len(result) > 0 {
		cli.current = result[0]
	}
	cli.mu.Unlock()
}

func (cli *CLI) render() {
	cli.mu.Lock()
	defer cli.mu.Unlock()

	query := []rune(queryMarker)
	queryLen := len(query)
	queryLineHeight := 1

	cli.Screen.Clear()

	for i, r := range []rune(query) {
		cli.Screen.SetContent(i, 0, r, nil, tcell.StyleDefault)
	}
	for i, r := range cli.inputRunes {
		cli.Screen.SetContent(i+queryLen+queryLineHeight, 0, r, nil, tcell.StyleDefault)
	}

	_, y := cli.Screen.Size()
	matchedLinesHeight := y - queryLineHeight

	for i := 0; i < matchedLinesHeight; i++ {
		if i > len(cli.matched)-1 {
			break
		} else {
			ma := cli.matched[i]
			for x, r := range []rune(ma.line) {
				cli.Screen.SetContent(x, i+queryLineHeight, r, nil, tcell.StyleDefault)
			}
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

			if key == tcell.KeyBackspace || key == tcell.KeyBackspace2 || key == tcell.KeyDelete {
				go cli.backspace()
				continue
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

func (cli *CLI) backspace() {
	if len(cli.inputRunes) == 0 {
		return
	}

	cli.mu.Lock()
	cli.inputRunes = cli.inputRunes[:len(cli.inputRunes)-1]
	cli.mu.Unlock()

	cli.doFilter()
	cli.render()
}

func (cli *CLI) exit() {
	cli.Screen.Fini()
}
