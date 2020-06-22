package main

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
