package fy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
)

type filterResult struct {
	matched []string
}

type filter interface {
	Run(context context.Context, reader io.ReadSeeker, resultCh chan<- filterResult) error
}

type cliFilter struct{}

func (f *cliFilter) Run(context context.Context, reader io.ReadSeeker, resultCh chan<- filterResult) error {
	var err error

	_, err = reader.Seek(0, 0) // todo
	if err != nil {
		return err
	}

	done := make(chan struct{})
	result := filterResult{matched: []string{}}

	go func() {
		defer func() { done <- struct{}{} }()

		// scanning...
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			result.matched = append(result.matched, text)
		}
		if serr := scanner.Err(); serr != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", serr)
			err = serr
			return
		}
	}()

	// todo: use context
	<-done
	resultCh <- result

	return err
}
