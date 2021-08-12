package main

import (
	"github.com/anishathalye/periscope/herror"

	"fmt"

	"github.com/dustin/go-humanize"
)

type optionPath struct {
	valid bool
	value string
}

func (op *optionPath) Set(x string) error {
	op.valid = true
	op.value = x
	return nil
}

func (op *optionPath) String() string {
	return op.value
}

func (op *optionPath) Type() string {
	return "path"
}

type size struct {
	value uint64
}

func (s *size) Set(x string) error {
	n, err := humanize.ParseBytes(x)
	if err != nil {
		return herror.UserF(nil, "cannot parse as a number of bytes")
	}
	s.value = n
	return nil
}

func (s *size) String() string {
	return fmt.Sprintf("%d", s.value)
}

func (s *size) Type() string {
	return "size"
}
