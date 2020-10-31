package fy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
)

type FilterResult struct {
	Error   error
	Matched []string
}

type Filter interface {
	Run(context context.Context, reader io.ReadSeeker, resultCh chan<- *FilterResult) error
	Quit()
}

type FilterImpl struct{}

func (f *FilterImpl) Run(context context.Context, reader io.ReadSeeker, resultCh chan<- *FilterResult) error {
	_, err := reader.Seek(0, 0)
	if err != nil {
		return err
	}

	done := make(chan struct{})
	result := &FilterResult{Matched: []string{}}

	go func() {
		defer func() { done <- struct{}{} }()

		// scanning...
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			result.Matched = append(result.Matched, text)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			return
		}
	}()

	// todo: use context
	<-done
	resultCh <- result

	return nil
}

func (f *FilterImpl) Quit() {
}
