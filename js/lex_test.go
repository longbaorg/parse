package js

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/tdewolff/test"
)

type TTs []TokenType

func TestTokens(t *testing.T) {
	var tokenTests = []struct {
		js       string
		expected []TokenType
	}{
		{" \t\v\f\u00A0\uFEFF\u2000", TTs{}}, // WhitespaceToken
		{"\n\r\r\n\u2028\u2029", TTs{LineTerminatorToken}},
		{"5.2 .04 0x0F 5e99", TTs{NumericToken, NumericToken, NumericToken, NumericToken}},
		{"a = 'string'", TTs{IdentifierToken, PunctuatorToken, StringToken}},
		{"/*comment*/ //comment", TTs{SingleLineCommentToken, SingleLineCommentToken}},
		{"{ } ( ) [ ]", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{". ; , < > <=", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{">= == != === !==", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"+ - * % ++ --", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"<< >> >>> & | ^", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"! ~ && || ? :", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"= += -= *= %= <<=", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{">>= >>>= &= |= ^= =>", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		//{"a = /.*/g;", TTs{IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken}},

		{"/*co\nm\u2028m/*ent*/ //co//mment\u2029//comment", TTs{MultiLineCommentToken, SingleLineCommentToken, LineTerminatorToken, SingleLineCommentToken}},
		{"<!-", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"1<!--2\n", TTs{NumericToken, SingleLineCommentToken, LineTerminatorToken}},
		{"x=y-->10\n", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, PunctuatorToken, NumericToken, LineTerminatorToken}},
		{"  /*comment*/ -->nothing\n", TTs{SingleLineCommentToken, PunctuatorToken, PunctuatorToken, IdentifierToken, LineTerminatorToken}},
		{"1 /*comment\nmultiline*/ -->nothing\n", TTs{NumericToken, MultiLineCommentToken, SingleLineCommentToken, LineTerminatorToken}},
		{"$ _\u200C \\u2000 \u200C", TTs{IdentifierToken, IdentifierToken, IdentifierToken, UnknownToken}},
		{">>>=>>>>=", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"1/", TTs{NumericToken, PunctuatorToken}},
		{"1/=", TTs{NumericToken, PunctuatorToken}},
		{"010xF", TTs{NumericToken, NumericToken, IdentifierToken}},
		{"50e+-0", TTs{NumericToken, IdentifierToken, PunctuatorToken, PunctuatorToken, NumericToken}},
		{"'str\\i\\'ng'", TTs{StringToken}},
		{"'str\\\\'abc", TTs{StringToken, IdentifierToken}},
		{"'str\\\ni\\\\u00A0ng'", TTs{StringToken}},
		//{"a = /[a-z/]/g", TTs{IdentifierToken, PunctuatorToken, RegexpToken}},
		//{"a=/=/g1", TTs{IdentifierToken, PunctuatorToken, RegexpToken}},
		//{"a = /'\\\\/\n", TTs{IdentifierToken, PunctuatorToken, RegexpToken, LineTerminatorToken}},
		//{"a=/\\//g1", TTs{IdentifierToken, PunctuatorToken, RegexpToken}},
		//{"new RegExp(a + /\\d{1,2}/.source)", TTs{IdentifierToken, IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},

		{"0b0101 0o0707 0b17", TTs{NumericToken, NumericToken, NumericToken, NumericToken}},
		{"`template`", TTs{TemplateToken}},
		{"`a${x+y}b`", TTs{TemplateToken, IdentifierToken, PunctuatorToken, IdentifierToken, TemplateToken}},
		{"`temp\nlate`", TTs{TemplateToken}},
		{"`outer${{x: 10}}bar${ raw`nested${2}endnest` }end`", TTs{TemplateToken, PunctuatorToken, IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, TemplateToken, IdentifierToken, TemplateToken, NumericToken, TemplateToken, TemplateToken}},
		{"`tmpl ${ a ? '' : `tmpl2 ${b ? 'b' : 'c'}` }`", TTs{TemplateToken, IdentifierToken, PunctuatorToken, StringToken, PunctuatorToken, TemplateToken, IdentifierToken, PunctuatorToken, StringToken, PunctuatorToken, StringToken, TemplateToken, TemplateToken}},

		// early endings
		{"'string", TTs{UnknownToken, IdentifierToken}},
		{"'\n '\u2028", TTs{UnknownToken, LineTerminatorToken, UnknownToken, LineTerminatorToken}},
		{"'str\\\U00100000ing\\0'", TTs{StringToken}},
		{"'strin\\00g'", TTs{StringToken}},
		{"/*comment", TTs{SingleLineCommentToken}},
		{"a=/regexp", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, IdentifierToken}},
		{"\\u002", TTs{UnknownToken, IdentifierToken}},

		// null characters
		{"'string\x00'return", TTs{StringToken, IdentifierToken}},
		{"//comment\x00comment\nreturn", TTs{SingleLineCommentToken, LineTerminatorToken, IdentifierToken}},
		{"/*comment\x00*/return", TTs{SingleLineCommentToken, IdentifierToken}},
		//{"a=/regexp\x00/;return", TTs{IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken, IdentifierToken}},
		//{"a=/regexp\\\x00/;return", TTs{IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken, IdentifierToken}},
		{"`template\x00`return", TTs{TemplateToken, IdentifierToken}},
		{"`template\\\x00`return", TTs{TemplateToken, IdentifierToken}},

		// coverage
		{"Ø a〉", TTs{IdentifierToken, IdentifierToken, UnknownToken}},
		{"0xg 0.f", TTs{NumericToken, IdentifierToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"0bg 0og", TTs{NumericToken, IdentifierToken, NumericToken, IdentifierToken}},
		{"\u00A0\uFEFF\u2000", TTs{}},
		{"\u2028\u2029", TTs{LineTerminatorToken}},
		{"\\u0029ident", TTs{IdentifierToken}},
		{"\\u{0029FEF}ident", TTs{IdentifierToken}},
		{"\\u{}", TTs{UnknownToken, IdentifierToken, PunctuatorToken, PunctuatorToken}},
		{"\\ugident", TTs{UnknownToken, IdentifierToken}},
		{"'str\u2028ing'", TTs{UnknownToken, IdentifierToken, LineTerminatorToken, IdentifierToken, UnknownToken}},
		{"a=/\\\n", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, UnknownToken, LineTerminatorToken}},
		//{"a=/x/\u200C\u3009", TTs{IdentifierToken, PunctuatorToken, RegexpToken, UnknownToken}},
		{"a=/x\n", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, IdentifierToken, LineTerminatorToken}},

		//{"return /abc/;", TTs{IdentifierToken, RegexpToken, PunctuatorToken}},
		//{"yield /abc/;", TTs{IdentifierToken, RegexpToken, PunctuatorToken}},
		{"a/b/g", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, IdentifierToken}},
		//{"{}/1/g", TTs{PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"i(0)/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		//{"if(0)/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, RegexpToken}},
		{"a.if(0)/1/g", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		//{"while(0)/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, RegexpToken}},
		//{"for(;;)/1/g", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		//{"with(0)/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, RegexpToken}},
		{"this/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		//{"case /1/g:", TTs{IdentifierToken, RegexpToken, PunctuatorToken}},
		//{"function f(){}/1/g", TTs{IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"this.return/1/g", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"(a+b)/1/g", TTs{PunctuatorToken, IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		//{"f(); function foo() {} /42/i", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"x = function() {} /42/i", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"x = function foo() {} /42/i", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		//{"x = /foo/", TTs{IdentifierToken, PunctuatorToken, RegexpToken}},
		{"x = x / foo /", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},
		//{"x = (/foo/)", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, RegexpToken, PunctuatorToken}},
		//{"x = 10 {/foo/}", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, RegexpToken, PunctuatorToken}},
		//{"do { /foo/ }", TTs{IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken}},
		//{"if (true) /foo/", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, RegexpToken}},
		{"x = (a) / foo", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, IdentifierToken, PunctuatorToken, PunctuatorToken, IdentifierToken}},
		{"bar (true) /foo/", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},
		{"`\\``", TTs{TemplateToken}},
		{"`\\${ 1 }`", TTs{TemplateToken}},
		{"`\\\r\n`", TTs{TemplateToken}},

		// go fuzz
		{"`", TTs{TemplateToken}},
	}

	for _, tt := range tokenTests {
		t.Run(tt.js, func(t *testing.T) {
			l := NewLexer(bytes.NewBufferString(tt.js))
			i := 0
			tokens := []TokenType{}
			for {
				token, _ := l.Next()
				if token == ErrorToken {
					test.T(t, l.Err(), io.EOF)
					break
				} else if token == WhitespaceToken {
					continue
				}
				tokens = append(tokens, token)
				i++
			}
			test.T(t, tokens, tt.expected, "token types must match")
		})
	}

	// coverage
	for i := 0; ; i++ {
		if TokenType(i).String() == fmt.Sprintf("Invalid(%d)", i) {
			break
		}
	}
}

func TestOffset(t *testing.T) {
	l := NewLexer(bytes.NewBufferString(`var i=5;`))
	test.T(t, l.Offset(), 0)
	_, _ = l.Next()
	test.T(t, l.Offset(), 3) // var
	_, _ = l.Next()
	test.T(t, l.Offset(), 4) // ws
	_, _ = l.Next()
	test.T(t, l.Offset(), 5) // i
	_, _ = l.Next()
	test.T(t, l.Offset(), 6) // =
	_, _ = l.Next()
	test.T(t, l.Offset(), 7) // 5
	_, _ = l.Next()
	test.T(t, l.Offset(), 8) // ;
}

////////////////////////////////////////////////////////////////

func ExampleNewLexer() {
	l := NewLexer(bytes.NewBufferString("var x = 'lorem ipsum';"))
	out := ""
	for {
		tt, data := l.Next()
		if tt == ErrorToken {
			break
		}
		out += string(data)
	}
	fmt.Println(out)
	// Output: var x = 'lorem ipsum';
}
