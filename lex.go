package purple

import (
	"fmt"
	"io"
	"unicode/utf8"
)

var (
	Digit      = &SetLexeme{"0123456789"}
	AlphaLower = &SetLexeme{"abcdefghiklmnopqrstuvwxyz"}
	AlphaUpper = &SetLexeme{"ABCDEFGHIJKLMNOPQRSTUVWXYZ"}
	Alpha      = &SetLexeme{"abcdefghiklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"}
	AlphaNum   = &SetLexeme{"0123456789abcdefghiklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"}
	Whitespace = &SetLexeme{" \t\n\f\r"}
)

func NewInputStream(data string) *InputStream {
	return &InputStream{0, []int{}, data}
}

type InputStream struct {
	offset    int
	markStack []int
	data      string
}

type Token struct {
	Lexeme
	Name string
}

func (is *InputStream) debug() string {
	return string(is.data[0:is.offset]) + "^" + string(is.data[is.offset:])
}

func (is *InputStream) markPush(pos int) {
	is.markStack = append(is.markStack, pos)
}

func (is *InputStream) markPop() int {
	if len(is.markStack) == 0 {
		panic("can't pop empty stack")
	}
	v := is.markStack[len(is.markStack)-1]
	is.markStack = is.markStack[0 : len(is.markStack)-1]
	return v
}

// Pushes a marker onto a stack. Calling Rewind() will pop it off the stack and reset the input
// stream to the position that was set when Mark() was called.
func (is *InputStream) Mark() {
	is.markPush(is.offset)
}

func (is *InputStream) Rewind() {
	is.offset = is.markPop()
}

func (is *InputStream) DiscardMark() {
	_ = is.markPop()
}

func (is InputStream) Peek(offset int) (rune, int, error) {
	if offset < 0 {
		panic("can't seek negative offset")
	}
	if is.offset >= len(is.data) {
		return 0, 0, io.EOF
	}

	var r rune
	w, width := 0, 0
	for i := 0; i <= offset; i++ {
		if is.offset+w >= len(is.data) {
			return 0, 0, io.EOF
		}
		r, width = utf8.DecodeRuneInString(is.data[is.offset+w:])
		w += width
	}
	return r, width, nil
}

func (is *InputStream) Consume() (rune, int, error) {
	if is.offset >= len(is.data) {
		return 0, 0, io.EOF
	}

	r, width := utf8.DecodeRuneInString(is.data[is.offset:])
	is.offset += width
	return r, width, nil
}

func (is *InputStream) Advance(i int) error {
	if i < 0 {
		panic("can't advance by negative number")
	}
	for i > 0 {
		_, _, err := is.Consume()
		if err != nil {
			return err
		}
	}
	return nil
}

type Lexeme interface {
	Match(is *InputStream, consume bool) (int, error)
}

type SetLexeme struct {
	l string
}

func (sm *SetLexeme) Match(is *InputStream, consume bool) (int, error) {
	if !consume {
		is.Mark()
		defer is.Rewind()
	}

	b, w, err := is.Consume()
	if err != nil {
		return -1, err
	}
	matched := -1
	for _, v := range sm.l {
		if v == b {
			matched = w
			break
		}
	}
	return matched, nil
}

type Or struct {
	m []Lexeme
}

func (o *Or) Match(is *InputStream, consume bool) (int, error) {
	for _, mtch := range o.m {
		if !consume {
			is.Mark()
		}
		v, err := mtch.Match(is, consume)
		if consume {
			is.Rewind()
		}
		if err != nil {
			return -1, err
		}
		if v >= 0 {
			return v, nil
		}
	}
	return -1, nil

}

type Then struct {
	m []Lexeme
}

func (t *Then) Match(is *InputStream, consume bool) (matchedLength int, err error) {
	is.Mark()
	defer func() {
		if !consume || matchedLength < 0 {
			is.Rewind()
		} else {
			is.DiscardMark()
		}
	}()
	err = nil
	matchedLength = -1
	for _, m := range t.m {
		var v int
		v, err = m.Match(is, true)
		if v <= 0 || err != nil {
			matchedLength = -1
			return
		}
		if matchedLength < 0 {
			matchedLength = 0
		}
		matchedLength += v
	}
	return

}

type Optional struct {
	l Lexeme
}

func (opt Optional) Match(is *InputStream, consume bool) (int, error) {
	if !consume {
		is.Mark()
		defer is.Rewind()
	}
	m, err := opt.l.Match(is, true)
	if err != nil {
		return -1, err
	}
	if m <= 0 {
		return 0, nil
	} else {
		return m, nil
	}
}

type ZeroOrMore struct {
	l Lexeme
}

func (zom ZeroOrMore) Match(is *InputStream, consume bool) (int, error) {
	if !consume {
		is.Mark()
		defer is.Rewind()
	}
	v, m := 0, -1
	var err error
	for {
		m, err = zom.l.Match(is, true)
		if err != nil {
			return -1, err
		}
		if m >= 0 {
			v += m
		} else {
			break
		}
	}
	return v, nil
}

type OneOrMore struct {
	l Lexeme
}

func (oom OneOrMore) Match(is *InputStream, consume bool) (matchLen int, err error) {
	is.Mark()
	defer func() {
		if !consume || matchLen < 0 {
			fmt.Println("rewinding")
			is.Rewind()
		} else {

			fmt.Println("discarding")
			is.DiscardMark()
		}
	}()

	matchLen = -1
	err = nil

	m := -1
	for {
		m, err = oom.l.Match(is, true)
		if err != nil {
			return -1, err
		}
		if m >= 0 {
			fmt.Println("matched")
			if matchLen < 0 {
				matchLen = 0
			}
			matchLen += m
		} else {
			fmt.Println("did not match, breaking")
			break
		}
	}

	fmt.Println("oneormore returning len", matchLen)
	return matchLen, nil
}

type And struct {
	m []Lexeme
}

type Literal struct {
	l string
}

func (l Literal) String() string {
	return fmt.Sprintf("Literal:'%v'", string(l.l))
}

func (lit Literal) Match(is *InputStream, consume bool) (int, error) {
	if !consume {
		is.Mark()
		defer is.Rewind()
	}

	for _, b := range lit.l {
		bi, _, err := is.Consume()
		if err != nil {
			return -1, err
		}
		if bi != b {
			return -1, nil
		}
	}
	return len(lit.l), nil
}
