package fy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
)

type matched struct {
	line  string
	index int
	pos1  int
	pos2  int
}

type filter interface {
	Run(context context.Context, input string, reader io.ReadSeeker, resultCh chan<- []matched) error
}

type cliFilter struct{}

func (f *cliFilter) Run(context context.Context, input string, reader io.ReadSeeker, resultCh chan<- []matched) error {
	var err error

	_, err = reader.Seek(0, 0) // todo
	if err != nil {
		return err
	}

	done := make(chan struct{})
	result := []matched{}

	go func() {
		defer func() { done <- struct{}{} }()

		scanner := bufio.NewScanner(reader)
		strs := []string{}

		for scanner.Scan() {
			strs = append(strs, scanner.Text())
		}
		if serr := scanner.Err(); serr != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", serr)
			err = serr
			return
		}

		matches, serr := findMatches(input, strs)
		if serr != nil {
			err = serr
		}

		result = matches
	}()

	// todo: use context
	<-done
	resultCh <- result

	return err
}

func findMatches(input string, lines []string) ([]matched, error) {
	var tmp []matched

	if len(input) == 0 {
		tmp = make([]matched, len(lines))
		for n, l := range lines {
			tmp[n] = matched{
				line:  l,
				index: n,
			}
		}
	} else {
		// https://github.com/mattn/gof/blob/192d3db8502dca43439c3c61ef957869a88e38b9/main.go#L106

		pat := "(?i)(?:.*)("
		for _, r := range input {
			pat += regexp.QuoteMeta(string(r)) + ".*?"
		}
		pat += ")"
		re := regexp.MustCompile(pat)

		tmp = make([]matched, 0, len(lines))

		for _, f := range lines {
			ms := re.FindAllStringSubmatchIndex(f, 1)
			if len(ms) != 1 || len(ms[0]) != 4 {
				continue
			}
			tmp = append(tmp, matched{
				line:  f,
				pos1:  len([]rune(f[0:ms[0][2]])),
				pos2:  len([]rune(f[0:ms[0][3]])),
				index: len(tmp),
			})
		}
	}

	return tmp, nil
}
