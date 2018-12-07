package terrors

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

type Type int

const (
	TypeUnknown Type = iota
	TypeInvalid
	TypePermission
	TypeExist
	TypeNotExist
	TypeInternal
	TypeUnauthorized
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func New(t Type, msg string) error {
	stack := errors.New(msg).(stackTracer).StackTrace()
	return &fundamental{t: t, msg: msg, stack: stack[1:]}
}

func Errorf(t Type, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	stack := errors.New(msg).(stackTracer).StackTrace()
	return &fundamental{t: t, msg: msg, stack: stack[1:]}
}

type fundamental struct {
	t     Type
	msg   string
	stack errors.StackTrace
}

func (f *fundamental) Type() Type {
	return f.t
}

func (f *fundamental) Error() string {
	return f.msg
}

func (f *fundamental) StackTrace() errors.StackTrace {
	return f.stack
}

func (f *fundamental) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, f.msg)
			f.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, f.msg)
	case 'q':
		fmt.Fprintf(s, "%q", f.msg)
	}
}

func WithStack(t Type, err error) error {
	if err == nil {
		return nil
	}
	stack := errors.New("").(stackTracer).StackTrace()
	return &withStack{t: t, cause: err, stack: stack[1:]}
}

type withStack struct {
	t     Type
	cause error
	stack errors.StackTrace
}

func (w *withStack) Type() Type {
	return w.t
}

func (w *withStack) Error() string {
	return w.cause.Error()
}

func (w *withStack) Cause() error {
	return w.cause
}

func (w *withStack) StackTrace() errors.StackTrace {
	return w.stack
}

func (w *withStack) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v", w.Cause())
			w.StackTrace().Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	}
}

func Wrap(t Type, err error, msg string) error {
	if err == nil {
		return nil
	}
	stack := errors.New("").(stackTracer).StackTrace()
	return &withStack{t: t, cause: &withMessage{t: t, cause: err, msg: msg}, stack: stack[1:]}
}

func Wrapf(t Type, err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	stack := errors.New("").(stackTracer).StackTrace()
	return &withStack{t: t, cause: &withMessage{t: t, cause: err, msg: fmt.Sprintf(format, args...)}, stack: stack[1:]}
}

func WithMessage(t Type, err error, message string) error {
	if err == nil {
		return nil
	}
	return &withMessage{t: t, cause: err, msg: message}
}

type withMessage struct {
	t     Type
	cause error
	msg   string
}

func (w *withMessage) Type() Type {
	return w.t
}

func (w *withMessage) Error() string {
	return w.msg + ": " + w.cause.Error()
}

func (w *withMessage) Cause() error {
	return w.cause
}

func (w *withMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", w.Cause())
			io.WriteString(s, w.msg)
			return
		}
		fallthrough
	case 's', 'q':
		io.WriteString(s, w.Error())
	}
}

func Cause(err error) error {
	type causer interface {
		Cause() error
	}

	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return err
}

func TypeOf(err error) Type {
	type typer interface {
		Type() Type
	}

	e, ok := err.(typer)
	if !ok {
		return TypeUnknown
	}

	return e.Type()
}
