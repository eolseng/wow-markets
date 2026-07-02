package luasv

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

type Value any

type Table struct {
	Fields  map[string]Value
	Indexed map[int]Value
	Array   []Value
}

func ParseVariable(reader io.Reader, variableName string) (*Table, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read SavedVariables: %w", err)
	}

	parser, err := newParser(string(content))
	if err != nil {
		return nil, err
	}

	value, err := parser.parseDocument(variableName)
	if err != nil {
		return nil, err
	}

	table, ok := value.(*Table)
	if !ok {
		return nil, fmt.Errorf("variable %s is not a table", variableName)
	}
	return table, nil
}

func (t *Table) Field(name string) (Value, bool) {
	if t.Fields == nil {
		return nil, false
	}
	value, ok := t.Fields[name]
	return value, ok
}

func (t *Table) Sequence() ([]Value, error) {
	if len(t.Indexed) == 0 {
		return t.Array, nil
	}

	valuesByIndex := make(map[int]Value, len(t.Array)+len(t.Indexed))
	for index, value := range t.Array {
		valuesByIndex[index+1] = value
	}
	for index, value := range t.Indexed {
		if _, exists := valuesByIndex[index]; exists {
			return nil, fmt.Errorf("table sequence contains duplicate index %d", index)
		}
		valuesByIndex[index] = value
	}

	indexes := make([]int, 0, len(valuesByIndex))
	for index := range valuesByIndex {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	values := make([]Value, 0, len(indexes))
	for expected, index := range indexes {
		if index != expected+1 {
			return nil, fmt.Errorf("table sequence is missing index %d", expected+1)
		}
		values = append(values, valuesByIndex[index])
	}
	return values, nil
}

type parser struct {
	lexer   *lexer
	current token
	peek    token
}

func newParser(input string) (*parser, error) {
	lexer := newLexer(input)
	current, err := lexer.next()
	if err != nil {
		return nil, err
	}
	peek, err := lexer.next()
	if err != nil {
		return nil, err
	}
	return &parser{lexer: lexer, current: current, peek: peek}, nil
}

func (p *parser) parseDocument(variableName string) (Value, error) {
	for p.current.kind != tokenEOF {
		if p.current.kind != tokenIdentifier {
			return nil, p.unexpected("top-level variable name")
		}

		name := p.current.text
		if err := p.advance(); err != nil {
			return nil, err
		}
		if err := p.consume(tokenEqual, "="); err != nil {
			return nil, err
		}

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		if name == variableName {
			return value, nil
		}

		if p.current.kind == tokenSemicolon {
			if err := p.advance(); err != nil {
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("variable %s not found", variableName)
}

func (p *parser) parseValue() (Value, error) {
	current := p.current

	switch current.kind {
	case tokenString:
		if err := p.advance(); err != nil {
			return nil, err
		}
		return current.text, nil
	case tokenNumber:
		if err := p.advance(); err != nil {
			return nil, err
		}
		if !strings.ContainsAny(current.text, ".eE") {
			value, err := strconv.ParseInt(current.text, 10, 64)
			if err != nil {
				return nil, fmt.Errorf(
					"line %d, column %d: invalid integer %q",
					current.line,
					current.column,
					current.text,
				)
			}
			return value, nil
		}
		value, err := strconv.ParseFloat(current.text, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"line %d, column %d: invalid number %q",
				current.line,
				current.column,
				current.text,
			)
		}
		return value, nil
	case tokenTrue:
		if err := p.advance(); err != nil {
			return nil, err
		}
		return true, nil
	case tokenFalse:
		if err := p.advance(); err != nil {
			return nil, err
		}
		return false, nil
	case tokenNil:
		if err := p.advance(); err != nil {
			return nil, err
		}
		return nil, nil
	case tokenLeftBrace:
		return p.parseTable()
	default:
		return nil, p.unexpected("value")
	}
}

func (p *parser) parseTable() (*Table, error) {
	if err := p.consume(tokenLeftBrace, "{"); err != nil {
		return nil, err
	}

	table := &Table{}

	for p.current.kind != tokenRightBrace {
		if p.current.kind == tokenEOF {
			return nil, p.unexpected("}")
		}

		switch {
		case p.current.kind == tokenLeftBracket:
			key, err := p.parseBracketKey()
			if err != nil {
				return nil, err
			}
			if err := p.consume(tokenEqual, "="); err != nil {
				return nil, err
			}
			value, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			if err := assignTableValue(table, key, value); err != nil {
				return nil, err
			}
		case p.current.kind == tokenIdentifier && p.peek.kind == tokenEqual:
			key := p.current.text
			if err := p.advance(); err != nil {
				return nil, err
			}
			if err := p.consume(tokenEqual, "="); err != nil {
				return nil, err
			}
			value, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			if table.Fields == nil {
				table.Fields = map[string]Value{}
			}
			table.Fields[key] = value
		default:
			value, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			table.Array = append(table.Array, value)
		}

		if p.current.kind == tokenComma || p.current.kind == tokenSemicolon {
			if err := p.advance(); err != nil {
				return nil, err
			}
		}
	}

	if err := p.consume(tokenRightBrace, "}"); err != nil {
		return nil, err
	}
	return table, nil
}

func (p *parser) parseBracketKey() (Value, error) {
	if err := p.consume(tokenLeftBracket, "["); err != nil {
		return nil, err
	}

	var key Value
	switch p.current.kind {
	case tokenString:
		key = p.current.text
	case tokenNumber:
		value, err := strconv.Atoi(p.current.text)
		if err != nil || value < 1 {
			return nil, fmt.Errorf(
				"line %d, column %d: invalid table index %q",
				p.current.line,
				p.current.column,
				p.current.text,
			)
		}
		key = value
	default:
		return nil, p.unexpected("string or positive integer table key")
	}

	if err := p.advance(); err != nil {
		return nil, err
	}
	if err := p.consume(tokenRightBracket, "]"); err != nil {
		return nil, err
	}
	return key, nil
}

func assignTableValue(table *Table, key Value, value Value) error {
	switch typed := key.(type) {
	case string:
		if table.Fields == nil {
			table.Fields = map[string]Value{}
		}
		table.Fields[typed] = value
	case int:
		if table.Indexed == nil {
			table.Indexed = map[int]Value{}
		}
		table.Indexed[typed] = value
	default:
		return fmt.Errorf("unsupported table key type %T", key)
	}
	return nil
}

func (p *parser) consume(kind tokenKind, description string) error {
	if p.current.kind != kind {
		return p.unexpected(description)
	}
	return p.advance()
}

func (p *parser) advance() error {
	p.current = p.peek
	next, err := p.lexer.next()
	if err != nil {
		return err
	}
	p.peek = next
	return nil
}

func (p *parser) unexpected(expected string) error {
	return fmt.Errorf(
		"line %d, column %d: expected %s, got %q",
		p.current.line,
		p.current.column,
		expected,
		p.current.text,
	)
}
