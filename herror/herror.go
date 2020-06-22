package herror

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
)

const maxStackDepth = 50

type Interface interface {
	error
	Herror(debug bool) string
}

type base struct {
	stackTrace string
}

func newBase() base {
	stackBuf := make([]uintptr, maxStackDepth)
	length := runtime.Callers(3, stackBuf[:])
	stack := stackBuf[:length]
	frames := runtime.CallersFrames(stack)
	buf := new(bytes.Buffer)
	for {
		frame, more := frames.Next()
		fn := frame.Function
		if fn == "" {
			fn = "???"
		}
		fmt.Fprintf(buf, "%s\n\t%s:%d\n", fn, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return base{buf.String()}
}

type silent struct {
	base
}

func (s *silent) Error() string {
	return ""
}

func (s *silent) Herror(debug bool) string {
	if !debug {
		return ""
	}
	return fmt.Sprintf("<silent error>\n--------\n%s--------\n", s.stackTrace)
}

func Silent() Interface {
	return &silent{base: newBase()}
}

func IsSilent(e Interface) bool {
	_, ok := e.(*silent)
	return ok
}

type user struct {
	base
	err     error
	message string
}

func (u *user) Error() string {
	if u.err != nil {
		return fmt.Sprintf("%s (%s)", u.message, u.err.Error())
	}
	return u.message
}

func (u *user) Herror(debug bool) string {
	buf := new(bytes.Buffer)
	fmt.Fprint(buf, u.message)
	if u.err != nil {
		fmt.Fprintf(buf, " (%s)", u.err.Error())
	}
	fmt.Fprintf(buf, "\n")
	if debug {
		fmt.Fprintf(buf, "--------\n%s--------\n", u.stackTrace)
	}
	return buf.String()
}

func User(err error, message string) Interface {
	return &user{base: newBase(), err: err, message: strings.TrimSpace(message)}
}

func UserF(err error, message string, a ...interface{}) Interface {
	return User(err, fmt.Sprintf(message, a...))
}

type unlikely struct {
	base
	err   error
	short string
	long  string
}

func (u *unlikely) Error() string {
	if u.err != nil {
		return fmt.Sprintf("%s (%s)", u.short, u.err.Error())
	}
	return u.short
}

func (u *unlikely) Herror(debug bool) string {
	buf := new(bytes.Buffer)
	fmt.Fprint(buf, u.short)
	if u.err != nil {
		fmt.Fprintf(buf, " (%s)", u.err.Error())
	}
	fmt.Fprint(buf, "\n")
	if len(u.long) > 0 {
		fmt.Fprintf(buf, "\n%s\n", u.long)
	}
	if debug {
		fmt.Fprintf(buf, "--------\n%s--------\n", u.stackTrace)
	}
	return buf.String()
}

func Unlikely(err error, short string, long string) Interface {
	return &unlikely{base: newBase(), short: strings.TrimSpace(short), err: err, long: strings.TrimSpace(long)}
}

type internal struct {
	base
	err     error
	message string
}

func (i *internal) Error() string {
	if i.message != "" {
		return fmt.Sprintf("%s (%s)", i.message, i.err.Error())
	}
	return i.err.Error()
}

func (i *internal) Herror(debug bool) string {
	buf := new(bytes.Buffer)
	fmt.Fprint(buf, "internal error: ")
	if i.message != "" {
		fmt.Fprintf(buf, "%s (%s)\n", i.message, i.err.Error())
	} else {
		fmt.Fprintf(buf, "%s\n", i.err.Error())
	}
	if !debug {
		fmt.Fprintf(buf, "\nThis might be a bug in periscope.\n\nRun with --debug to see a stack trace, and please consider opening a GitHub issue to report this occurrence.\n")
	} else {
		fmt.Fprintf(buf, "--------\n%s--------\n", i.stackTrace)
	}
	return buf.String()
}

func Internal(err error, message string) Interface {
	return &internal{base: newBase(), message: strings.TrimSpace(message), err: err}
}
