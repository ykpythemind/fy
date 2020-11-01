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
	"github.com/mattn/go-isatty"
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
	screen     tcell.Screen
	mu         sync.Mutex
	debug      bool
	needClose  bool
}

func New(args []string, debug bool) (*CLI, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}

	var in io.Reader
	needClose := false

	if len(args) == 2 {
		f, err := os.Open(args[1])
		if err != nil {
			return nil, err
		}
		in = f
		needClose = true
	} else if !isatty.IsTerminal(os.Stdin.Fd()) {
		in = os.Stdin
	} else {
		return nil, errors.New("you must supply something to work with via stdin or filename")
	}

	// todo: readallやめる
	by, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(by)
	// ---

	if err := screen.Init(); err != nil {
		return nil, err
	}

	filter := &cliFilter{}

	cli := &CLI{
		input:     reader,
		output:    os.Stdout, // wip
		keyCh:     make(chan rune),
		screen:    screen,
		quitCh:    make(chan struct{}),
		filterCh:  make(chan []matched, 1),
		selectCh:  make(chan struct{}),
		current:   matched{},
		filter:    filter,
		matched:   []matched{},
		debug:     debug,
		needClose: needClose,
	}
	return cli, nil
}

func (cli *CLI) Run() error {
	go cli.handleEvent()
	go cli.handleKeyInput()
	go cli.filterInput()

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
		fmt.Println()
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

		go cli.filterInput()
	}
}

func (cli *CLI) filterInput() {
	context := context.TODO()

	// フィルターをキャンセルできるようにしたいけど...
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
	cli.render()
}

func (cli *CLI) render() {
	cli.mu.Lock()
	defer cli.mu.Unlock()

	query := []rune(queryMarker)
	queryLen := len(query)
	queryLineHeight := 1

	cli.screen.Clear()

	// query line
	for i, r := range []rune(query) {
		cli.screen.SetContent(i, 0, r, nil, tcell.StyleDefault)
	}
	for i, r := range cli.inputRunes {
		cli.screen.SetContent(i+queryLen+queryLineHeight, 0, r, nil, tcell.StyleDefault)
	}

	// matched lines
	wx, wy := cli.screen.Size()
	matchedLinesHeight := wy - queryLineHeight

	current := cli.current

	for i := 0; i < matchedLinesHeight; i++ {
		if i > len(cli.matched)-1 {
			break
		} else {
			ma := cli.matched[i]
			st := tcell.StyleDefault
			matchedline := false

			if ma.index == current.index {
				matchedline = true
				st = st.Background(tcell.NewRGBColor(200, 200, 0))
			}

			if matchedline {
				for i := 0; i < wx; i++ {
					cli.screen.SetContent(i, current.index+queryLineHeight, ' ', nil, st)
				}
			}

			for x, r := range []rune(ma.line) {
				s := st
				if ma.pos1 <= x && x < ma.pos2 {
					s = st.Bold(true)
				}
				cli.screen.SetContent(x, i+queryLineHeight, r, nil, s)
			}
		}
	}

	cli.screen.Show()
}

func (cli *CLI) handleEvent() {
	for {
		ev := cli.screen.PollEvent()

		switch ev := ev.(type) {
		case *tcell.EventKey:
			key := ev.Key()
			if key == tcell.KeyEscape {
				close(cli.quitCh)
				return
			}

			if key == tcell.KeyEnter {
				close(cli.selectCh)
				return
			}

			if key == tcell.KeyCtrlN {
				index := cli.current.index
				go cli.changeCurrent(index + 1)
				continue
			}

			if key == tcell.KeyCtrlP {
				index := cli.current.index
				go cli.changeCurrent(index - 1)
				continue
			}

			if key == tcell.KeyBackspace || key == tcell.KeyBackspace2 || key == tcell.KeyDelete {
				go cli.backspace()
				continue
			}

			if _, isSpecialKey := tcell.KeyNames[key]; isSpecialKey {
				// ignore
				continue
			}

			// 通常の入力
			r := ev.Rune()
			cli.keyCh <- r
		case *tcell.EventResize:
			cli.screen.Sync()
		}
	}
}

func (cli *CLI) changeCurrent(index int) {
	cli.mu.Lock()
	matched := cli.matched
	cli.mu.Unlock()

	if index > len(cli.matched)-1 || index < 0 {
		return // do nothing
	}

	ma := matched[index]
	cli.mu.Lock()
	cli.current = ma
	cli.mu.Unlock()

	cli.render()
}

func (cli *CLI) backspace() {
	if len(cli.inputRunes) == 0 {
		return
	}

	cli.mu.Lock()
	cli.inputRunes = cli.inputRunes[:len(cli.inputRunes)-1]
	cli.mu.Unlock()

	cli.filterInput()
	cli.render()
}

func (cli *CLI) exit() {
	cli.screen.Fini()

	if cli.needClose {
		if in, ok := cli.input.(io.Closer); ok {
			in.Close()
		}
	}
}
