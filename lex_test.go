package purple

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"testing"
)

func TestInputStream(t *testing.T) {
	Convey("With an inputstream", t, func() {
		i1 := NewInputStream("01234")

		v, w, err := i1.Peek(0)
		So(err, ShouldBeNil)
		So(w, ShouldEqual, 1)
		So(v, ShouldEqual, byte('0'))

		v, w, err = i1.Peek(4)
		So(err, ShouldBeNil)
		So(v, ShouldEqual, byte('4'))

		v, w, err = i1.Peek(5)
		So(err, ShouldNotBeNil)
		So(v, ShouldEqual, byte(0))

		for _, b := range []byte("01234") {
			v, w, err := i1.Consume()
			fmt.Println(string(v), string(b))
			So(err, ShouldBeNil)
			So(v, ShouldEqual, b)
			So(w, ShouldEqual, 1)
		}

		// Consume from empty input stream should return a zero-width
		v, w, err = i1.Consume()
		So(err, ShouldNotBeNil)
		So(w, ShouldEqual, 0)
		So(v, ShouldEqual, byte(0))

	})
}

func TestLiteral(t *testing.T) {
	Convey("With an inputstream", t, func() {
		i1 := NewInputStream("01234")
		l1 := &Literal{"0123"}

		m, err := l1.Match(i1, false)
		So(m, ShouldEqual, 4)
		So(err, ShouldBeNil)

		b, w, err := i1.Peek(0)
		So(b, ShouldEqual, '0')
		So(w, ShouldEqual, 1)
		So(err, ShouldBeNil)

		m, err = l1.Match(i1, true)
		So(m, ShouldEqual, 4)
		So(err, ShouldBeNil)
		b, w, err = i1.Peek(0)
		fmt.Println("got [", string(b), "]")
		So(b, ShouldEqual, '4')
		So(w, ShouldEqual, 1)
	})
}

func TestOr(t *testing.T) {
	Convey("With an inputstream", t, func() {
		i1 := NewInputStream("foo bar baz")
		i2 := NewInputStream("blah bar baz")
		i3 := NewInputStream("cow bar baz")
		o1 := &Or{[]Lexeme{&Literal{"foo"}, &Literal{"blah"}}}

		m, err := o1.Match(i1, false)
		So(m, ShouldEqual, 3)
		So(err, ShouldBeNil)

		m, err = o1.Match(i2, false)
		So(m, ShouldEqual, 4)
		So(err, ShouldBeNil)

		m, err = o1.Match(i3, false)
		So(m, ShouldEqual, -1)
		So(err, ShouldBeNil)
	})
}

func TestThen(t *testing.T) {
	Convey("With an inputstream", t, func() {
		i1 := NewInputStream("foo bar baz")
		i2 := NewInputStream("foo bah baz")
		//i3 := NewInputStream("cow bar baz")
		t1 := &Then{[]Lexeme{&Literal{"foo "}, &Literal{"bar"}}}
		m, err := t1.Match(i1, true)
		So(m, ShouldEqual, 7)
		So(err, ShouldBeNil)

		m, err = t1.Match(i2, true)
		So(m, ShouldEqual, -1)
		So(err, ShouldBeNil)
	})
}

func TestOneOrMore(t *testing.T) {
	Convey("With an inputstream", t, func() {
		i1 := NewInputStream("baz123")
		t1 := &OneOrMore{Alpha}
		m, err := t1.Match(i1, true)
		So(m, ShouldEqual, 3)
		So(err, ShouldBeNil)

		t2 := &OneOrMore{Digit}
		m, err = t2.Match(i1, true)
		So(m, ShouldEqual, -1)
	})
}

func TestZeroOrMore(t *testing.T) {
	Convey("With an inputstream", t, func() {
		i1 := NewInputStream("baz123")
		t1 := &ZeroOrMore{Alpha}
		m, err := t1.Match(i1, false)
		So(m, ShouldEqual, 3)
		So(err, ShouldBeNil)

		t2 := &ZeroOrMore{Digit}
		m, err = t2.Match(i1, false)
		So(err, ShouldBeNil)
		So(m, ShouldEqual, 0)
	})
}

func TestLexerTest(t *testing.T) {
	ts := []Token{
		{Literal{"("}, "LPAREN"},
		{Literal{")"}, "RPAREN"},
		{OneOrMore{Digit}, "number"},
		{&SetLexeme{"*+-/"}, "operator"},
		{ZeroOrMore{Whitespace}, "whitespace"},
	}

	is := NewInputStream("((1+2)/(5*9)) - 1")

outerloop:
	for {
		fmt.Println("isis: ", is.debug())
		fmt.Println("offset is", is.offset)
		v := 0
		var err error
		for _, t := range ts {
			v, err = t.Match(is, true)
			if err == io.EOF {
				break outerloop
			}
			if err != nil {
				panic(err)
			}
			if v <= 0 {
				continue
			}
			fmt.Println("got token", t.Name, v)
			fmt.Println("token text is", is.data[is.offset-v:is.offset])
			fmt.Println("offset is now", is.offset, "\n\n")
			continue outerloop
		}
		//is.offset += v
		panic("no match")
	}

}
