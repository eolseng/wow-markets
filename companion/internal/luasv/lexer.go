package luasv

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenIdentifier
	tokenNumber
	tokenString
	tokenTrue
	tokenFalse
	tokenNil
	tokenLeftBrace
	tokenRightBrace
	tokenLeftBracket
	tokenRightBracket
	tokenEqual
	tokenComma
	tokenSemicolon
)

type token struct {
	kind   tokenKind
	text   string
	line   int
	column int
}

type lexer struct {
	input  []rune
	offset int
	line   int
	column int
}

func newLexer(input string) *lexer {
	return &lexer{
		input:  []rune(input),
		line:   1,
		column: 1,
	}
}

func (l *lexer) next() (token, error) {
	l.skipIgnored()

	if l.offset >= len(l.input) {
		return token{kind: tokenEOF, line: l.line, column: l.column}, nil
	}

	startLine, startColumn := l.line, l.column
	current := l.input[l.offset]

	switch current {
	case '{':
		l.advance()
		return token{kind: tokenLeftBrace, text: "{", line: startLine, column: startColumn}, nil
	case '}':
		l.advance()
		return token{kind: tokenRightBrace, text: "}", line: startLine, column: startColumn}, nil
	case '[':
		l.advance()
		return token{kind: tokenLeftBracket, text: "[", line: startLine, column: startColumn}, nil
	case ']':
		l.advance()
		return token{kind: tokenRightBracket, text: "]", line: startLine, column: startColumn}, nil
	case '=':
		l.advance()
		return token{kind: tokenEqual, text: "=", line: startLine, column: startColumn}, nil
	case ',':
		l.advance()
		return token{kind: tokenComma, text: ",", line: startLine, column: startColumn}, nil
	case ';':
		l.advance()
		return token{kind: tokenSemicolon, text: ";", line: startLine, column: startColumn}, nil
	case '"', '\'':
		value, err := l.scanString(current)
		if err != nil {
			return token{}, fmt.Errorf("line %d, column %d: %w", startLine, startColumn, err)
		}
		return token{kind: tokenString, text: value, line: startLine, column: startColumn}, nil
	}

	if isIdentifierStart(current) {
		text := l.scanIdentifier()
		kind := tokenIdentifier
		switch text {
		case "true":
			kind = tokenTrue
		case "false":
			kind = tokenFalse
		case "nil":
			kind = tokenNil
		}
		return token{kind: kind, text: text, line: startLine, column: startColumn}, nil
	}

	if current == '-' || unicode.IsDigit(current) {
		text := l.scanNumber()
		if _, err := strconv.ParseFloat(text, 64); err != nil {
			return token{}, fmt.Errorf(
				"line %d, column %d: invalid number %q",
				startLine,
				startColumn,
				text,
			)
		}
		return token{kind: tokenNumber, text: text, line: startLine, column: startColumn}, nil
	}

	return token{}, fmt.Errorf(
		"line %d, column %d: unsupported character %q",
		startLine,
		startColumn,
		current,
	)
}

func (l *lexer) skipIgnored() {
	for l.offset < len(l.input) {
		if unicode.IsSpace(l.input[l.offset]) {
			l.advance()
			continue
		}

		if l.input[l.offset] == '-' &&
			l.offset+1 < len(l.input) &&
			l.input[l.offset+1] == '-' {
			for l.offset < len(l.input) && l.input[l.offset] != '\n' {
				l.advance()
			}
			continue
		}

		return
	}
}

func (l *lexer) scanIdentifier() string {
	start := l.offset
	for l.offset < len(l.input) && isIdentifierPart(l.input[l.offset]) {
		l.advance()
	}
	return string(l.input[start:l.offset])
}

func (l *lexer) scanNumber() string {
	start := l.offset
	if l.input[l.offset] == '-' {
		l.advance()
	}

	for l.offset < len(l.input) {
		current := l.input[l.offset]
		if unicode.IsDigit(current) ||
			current == '.' ||
			current == 'e' ||
			current == 'E' ||
			current == '+' ||
			current == '-' {
			l.advance()
			continue
		}
		break
	}

	return string(l.input[start:l.offset])
}

func (l *lexer) scanString(quote rune) (string, error) {
	l.advance()
	var output strings.Builder

	for l.offset < len(l.input) {
		current := l.input[l.offset]
		l.advance()

		if current == quote {
			return output.String(), nil
		}
		if current != '\\' {
			output.WriteRune(current)
			continue
		}
		if l.offset >= len(l.input) {
			return "", fmt.Errorf("unterminated escape sequence")
		}

		escaped := l.input[l.offset]
		l.advance()
		switch escaped {
		case 'a':
			output.WriteRune('\a')
		case 'b':
			output.WriteRune('\b')
		case 'f':
			output.WriteRune('\f')
		case 'n':
			output.WriteRune('\n')
		case 'r':
			output.WriteRune('\r')
		case 't':
			output.WriteRune('\t')
		case 'v':
			output.WriteRune('\v')
		case '\\', '"', '\'':
			output.WriteRune(escaped)
		default:
			if !unicode.IsDigit(escaped) {
				return "", fmt.Errorf("unsupported escape sequence \\%c", escaped)
			}

			digits := []rune{escaped}
			for len(digits) < 3 &&
				l.offset < len(l.input) &&
				unicode.IsDigit(l.input[l.offset]) {
				digits = append(digits, l.input[l.offset])
				l.advance()
			}
			value, err := strconv.ParseInt(string(digits), 10, 32)
			if err != nil || value > 255 {
				return "", fmt.Errorf("invalid decimal escape \\%s", string(digits))
			}
			output.WriteRune(rune(value))
		}
	}

	return "", fmt.Errorf("unterminated string")
}

func (l *lexer) advance() {
	if l.offset >= len(l.input) {
		return
	}

	if l.input[l.offset] == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	l.offset++
}

func isIdentifierStart(value rune) bool {
	return value == '_' || unicode.IsLetter(value)
}

func isIdentifierPart(value rune) bool {
	return isIdentifierStart(value) || unicode.IsDigit(value)
}
