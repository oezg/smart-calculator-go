package main

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const (
	UNKNOWN = "Unknown variable"
	INVALID = "Invalid"
	HELP    = `Smart calculator can save values to variables
and calculate the value of arithmetic expressions.
The supported operations are assignment '=', parenthesis '()', addition '+', 
subtraction '-', multiplication '*', integer division '/', 
modulo '%' and exponent '^'. 
Variable names can only have Latin characters but no digits or special characters.
Smart calculator works only with integers and not with floating point numbers.
Type "/exit" to end the program`
)

var memory = make(map[Identifier]Value)

type Identifier string

type Value int

type ValueStack []Value

type Operator string

type OperatorStack []Operator

type Expression []Term

type Term struct {
	Value      Value
	IsOperator bool
	Operator   Operator
}

type RawTerm struct {
	isIdentifier, isValue, isOperator, closed bool
	Text                                      string
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		handleCommand(scanner.Text())
	}
}

func handleCommand(text string) {
	if strings.HasPrefix(text, "/") {
		switch text[1:] {
		case "exit":
			fmt.Println("Bye!")
			os.Exit(0)
		case "help":
			fmt.Println(HELP)
		default:
			fmt.Println("Unknown command")
		}
	} else {
		handleAssignment(text)
	}
}

func handleAssignment(text string) {
	if strings.Contains(text, "=") {
		assignmentSides := strings.SplitN(text, "=", 2)
		assignee := strings.TrimSpace(assignmentSides[0])
		assigned := strings.TrimSpace(assignmentSides[1])
		if !isIdentifier(assignee) {
			fmt.Println("Invalid identifier")
		} else {
			expression, err := makeExpression(assigned)
			if err != nil {
				message := err.Error()
				if message == INVALID {
					message += " assignment"
				}
				fmt.Println(message)
			} else {
				memory[Identifier(assignee)], err = expression.Evaluate()
				if err != nil {
					message := err.Error()
					if message == INVALID {
						message += " assignment"
					}
					fmt.Println(message)
				}
			}
		}
	} else {
		handleExpression(text)
	}
}

func handleExpression(text string) {
	expression, err := makeExpression(text)
	if err != nil {
		message := err.Error()
		if message == INVALID {
			message += " expression"
		}
		fmt.Println(message)
	} else if !IsEmpty(expression) {
		result, err := expression.Evaluate()
		if err != nil {
			message := err.Error()
			if message == INVALID {
				message += " expression"
			}
			fmt.Println(message)
		} else {
			fmt.Println(result)
		}
	}
}

func (operator Operator) Operate(value1, value2 Value) (result Value, err error) {
	switch operator {
	case "+":
		result = value1 + value2
	case "-":
		result = value1 - value2
	case "*":
		result = value1 * value2
	case "/":
		result = value1 / value2
	case "%":
		result = value1 % value2
	case "^":
		result = Value(math.Pow(float64(value1), float64(value2)))
	default:
		err = errors.New(INVALID)
	}
	return
}

func (expression *Expression) Add(terms ...Term) {
	for _, term := range terms {
		*expression = Push(*expression, term)
	}
}

func (expression *Expression) Evaluate() (Value, error) {
	var stack ValueStack
	for _, term := range *expression {
		if term.IsOperator {
			tempStack, value1 := Pop(stack)
			stack = tempStack
			if stack == nil {
				return 0, errors.New(INVALID)
			}
			tempStack, value2 := Pop(stack)
			stack = tempStack
			if stack == nil {
				return 0, errors.New(INVALID)
			}
			if result, err := term.Operator.Operate(value2, value1); err == nil {
				stack = Push(stack, result)
			} else {
				return 0, err
			}
		} else {
			stack = Push(stack, term.Value)
		}
	}
	return Peek(stack), nil
}

func (expression *Expression) Grow(stack OperatorStack, term RawTerm) (OperatorStack, error) {
	if term.isOperator {
		if operator, ok := isOperator(term.Text); ok {
			poppedOperators, err := stack.Update(operator)
			if err != nil {
				return nil, errors.New(INVALID)
			}
			expression.Add(poppedOperators...)
		} else {
			return nil, errors.New(INVALID)
		}
	} else if term.isValue {
		if value, ok := isNumber(term.Text); ok {
			expression.Add(Term{Value: value})
		} else {
			return nil, errors.New(INVALID)
		}
	} else if term.isIdentifier {
		if value, ok := memory[Identifier(term.Text)]; ok {
			expression.Add(Term{Value: value})
		} else if isIdentifier(term.Text) {
			return nil, errors.New(UNKNOWN)
		} else {
			return nil, errors.New(INVALID)
		}
	}
	return stack, nil
}

func Precedence(operator Operator) (precedence int8) {
	switch operator {
	case "+", "-":
		precedence = 1
	case "*", "/", "%":
		precedence = 2
	case "^":
		precedence = 3
	}
	return
}

func IsEmpty[T comparable](list []T) bool {
	return len(list) == 0
}

func Peek[T comparable](stack []T) T {
	var t T
	if IsEmpty(stack) {
		return t
	}
	return stack[len(stack)-1]
}

func Push[T comparable](stack []T, element T) []T {
	return append(stack, element)
}

func Pop[T comparable](stack []T) ([]T, T) {
	if IsEmpty(stack) {
		var t T
		return nil, t
	}
	last := len(stack) - 1
	return stack[:last], stack[last]
}

func (stack *OperatorStack) Update(operator Operator) (operators []Term, err error) {
	if IsEmpty(*stack) || "(" == operator || "(" == Peek(*stack) {
		*stack = Push(*stack, operator)
		return
	}
	if ")" == operator {
		for !IsEmpty(*stack) {
			tempStack, topOfStack := Pop(*stack)
			*stack = tempStack
			if "(" == topOfStack {
				return
			} else {
				operators = Push(operators, Term{Operator: topOfStack, IsOperator: true})
			}
		}
		err = errors.New(INVALID)
		return
	}
	if Precedence(Peek(*stack)) < Precedence(operator) {
		*stack = Push(*stack, operator)
		return
	}
	for !IsEmpty(*stack) {
		topOfStack := Peek(*stack)
		if "(" == topOfStack || Precedence(topOfStack) < Precedence(operator) {
			*stack = Push(*stack, operator)
			return
		} else {
			*stack, topOfStack = Pop(*stack)
			operators = append(operators, Term{Operator: topOfStack, IsOperator: true})
		}
	}
	*stack = Push(*stack, operator)
	return
}

func (term *RawTerm) Close(last RawTerm, char string) {
	switch {
	case " " == char:
		term.closed = true
	case term.isValue:
		_, ok := isNumber(char)
		term.closed = !ok
	case term.isOperator:
		_, ok := isNumber(char)
		if ok && (term.Text == "+" || term.Text == "-") {
			term.closed = last.isIdentifier || last.isValue
		} else if strings.HasSuffix(term.Text, "+") || strings.HasSuffix(term.Text, "-") {
			term.closed = !(char == "+" || char == "-")
		} else {
			term.closed = true
		}
	case term.isIdentifier:
		term.closed = !isIdentifier(char)
	}
}

func (term *RawTerm) Extend(char string) (err error) {
	switch {
	case " " == char:
		return
	case term.isValue:
	case term.isOperator:
		if _, ok := isNumber(char); ok {
			term.isOperator = false
			term.isValue = true
		}
	case term.isIdentifier:
	default:
		if _, ok := isNumber(char); ok {
			term.isValue = true
		} else if _, ok = isOperator(char); ok {
			term.isOperator = true
		} else if isIdentifier(char) {
			term.isIdentifier = true
		} else {
			err = errors.New(INVALID)
		}
	}
	term.Text += char
	return
}

func makeExpression(text string) (expression Expression, err error) {
	reader := strings.NewReader(text)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanRunes)
	var stack OperatorStack
	var currentTerm, lastTerm RawTerm
	for scanner.Scan() {
		currentTerm.Close(lastTerm, scanner.Text())
		if currentTerm.closed {
			if stack, err = expression.Grow(stack, currentTerm); err != nil {
				return
			}
			lastTerm, currentTerm = currentTerm, RawTerm{}
		}
		if err = currentTerm.Extend(scanner.Text()); err != nil {
			return
		}
	}
	if stack, err = expression.Grow(stack, currentTerm); err != nil {
		return
	}
	for !IsEmpty(stack) {
		var operator Operator
		stack, operator = Pop(stack)
		expression.Add(Term{Operator: operator, IsOperator: true})
	}
	return
}

func isNumber(text string) (Value, bool) {
	number, err := strconv.Atoi(text)
	if err != nil {
		return Value(0), false
	}
	return Value(number), true
}

func isOperator(text string) (Operator, bool) {
	switch text {
	case "*", "/", "^", "%", "(", ")", "+", "-":
		return Operator(text), true
	}
	if minus, err := plusMinus(text); err == nil {
		return Operator(minus), true
	}
	return "", false
}

func isIdentifier(text string) bool {
	for _, char := range text {
		if !unicode.In(char, unicode.Latin) {
			return false
		}
	}
	return true
}

func plusMinus(text string) (string, error) {
	var negative bool
	for _, symbol := range text {
		if symbol == '+' {
			continue
		}
		if symbol == '-' {
			negative = !negative
		} else {
			return "", errors.New(INVALID)
		}
	}
	if negative {
		return "-", nil
	}
	return "+", nil
}
