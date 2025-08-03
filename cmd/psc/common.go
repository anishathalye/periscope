package main

import (
	"github.com/anishathalye/periscope/internal/herror"

	"fmt"
	"math"

	"github.com/dustin/go-humanize"
)

type size struct {
	value int64
}

func (s *size) Set(x string) error {
	n, err := humanize.ParseBytes(x)
	if err != nil {
		return herror.UserF(nil, "cannot parse as a number of bytes")
	}
	s.value = int64(n)
	if s.value < 0 {
		s.value = math.MaxInt64
	}
	return nil
}

func (s *size) String() string {
	return fmt.Sprintf("%d", s.value)
}

func (s *size) Type() string {
	return "size"
}
