package php8

import (
	"github.com/pyxsoft/php-parser/pkg/ast"
	"github.com/pyxsoft/php-parser/pkg/position"
	"github.com/pyxsoft/php-parser/pkg/token"
)

type ParserBrackets struct {
	Position        *position.Position
	OpenBracketTkn  *token.Token
	Child           ast.Vertex
	CloseBracketTkn *token.Token
}

func (n *ParserBrackets) Accept(v ast.Visitor) {
	// do nothing
}

func (n *ParserBrackets) GetPosition() *position.Position {
	return n.Position
}

type ParserSeparatedList struct {
	Position      *position.Position
	Items         []ast.Vertex
	SeparatorTkns []*token.Token
}

func (n *ParserSeparatedList) Accept(v ast.Visitor) {
	// do nothing
}

func (n *ParserSeparatedList) GetPosition() *position.Position {
	return n.Position
}

// TraitAdaptationList node
type TraitAdaptationList struct {
	Position             *position.Position
	OpenCurlyBracketTkn  *token.Token
	Adaptations          []ast.Vertex
	CloseCurlyBracketTkn *token.Token
}

func (n *TraitAdaptationList) Accept(v ast.Visitor) {
	// do nothing
}

func (n *TraitAdaptationList) GetPosition() *position.Position {
	return n.Position
}

// ArgumentList node
type ArgumentList struct {
	Position            *position.Position
	OpenParenthesisTkn  *token.Token
	Arguments           []ast.Vertex
	SeparatorTkns       []*token.Token
	EllipsisTkn         *token.Token
	CloseParenthesisTkn *token.Token
}

func (n *ArgumentList) Accept(v ast.Visitor) {
	// do nothing
}

func (n *ArgumentList) GetPosition() *position.Position {
	return n.Position
}

type EnumCaseExpr struct {
	Position  *position.Position
	AssignTkn *token.Token
	Expr      ast.Vertex
}

func (n *EnumCaseExpr) Accept(v ast.Visitor) {
	// do nothing
}

func (n *EnumCaseExpr) GetPosition() *position.Position {
	return n.Position
}

type ReturnType struct {
	Position *position.Position
	ColonTkn *token.Token
	Type     ast.Vertex
}

func (n *ReturnType) Accept(v ast.Visitor) {
	// do nothing
}

func (n *ReturnType) GetPosition() *position.Position {
	return n.Position
}

// TraitMethodRef node
type TraitMethodRef struct {
	Position       *position.Position
	Trait          ast.Vertex
	DoubleColonTkn *token.Token
	Method         ast.Vertex
}

func (n *TraitMethodRef) Accept(v ast.Visitor) {
	// do nothing
}

func (n *TraitMethodRef) GetPosition() *position.Position {
	return n.Position
}

// ParserParenthesizedList is a parenthesized, possibly empty, separated list
// (e.g. property hook parameters).
type ParserParenthesizedList struct {
	Position            *position.Position
	OpenParenthesisTkn  *token.Token
	List                ast.Vertex
	CloseParenthesisTkn *token.Token
}

func (n *ParserParenthesizedList) Accept(v ast.Visitor) {
	// do nothing
}

func (n *ParserParenthesizedList) GetPosition() *position.Position {
	return n.Position
}

// PropertyHookList is the `{ hook* }` block of a PHP 8.4 hooked property.
type PropertyHookList struct {
	Position             *position.Position
	OpenCurlyBracketTkn  *token.Token
	Hooks                []ast.Vertex
	CloseCurlyBracketTkn *token.Token
}

func (n *PropertyHookList) Accept(v ast.Visitor) {
	// do nothing
}

func (n *PropertyHookList) GetPosition() *position.Position {
	return n.Position
}

// PropertyHookBody holds one of the three property hook body forms.
type PropertyHookBody struct {
	Position       *position.Position
	DoubleArrowTkn *token.Token
	Expr           ast.Vertex
	Stmt           ast.Vertex
	SemiColonTkn   *token.Token
}

func (n *PropertyHookBody) Accept(v ast.Visitor) {
	// do nothing
}

func (n *PropertyHookBody) GetPosition() *position.Position {
	return n.Position
}
