package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const (
	UNKNOWN = "Unknown variable"
	INVALID = "Invalid"
	HELP    = `The program calculates the result of the arithmetic expression
The program can calculate addition or subtraction
The program saves values to variables using "=" operator
Type "/exit" to end the program`
)

var memory = make(map[Identifier]Value)

type Identifier string

type Value int

type ValueStack []Value

type Operator string

type OperatorStack []Operator

type Term struct {
	Value      Value
	IsOperator bool
	Operator   Operator
}

type Expression struct {
	Terms []Term
}

func (operator Operator) Operate(value1, value2 Value) (result Value) {
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
		log.Fatal(operator, "is not an operator")
	}
	return
}

func (expression *Expression) IsEmpty() bool {
	return len(expression.Terms) == 0
}

func (expression *Expression) Evaluate() Value {
	var stack ValueStack
	for _, term := range expression.Terms {
		if term.IsOperator {
			tempStack, value1 := Pop(stack)
			stack = tempStack
			if stack == nil {
				log.Fatal("Operator has no operands")
			}
			tempStack, value2 := Pop(stack)
			stack = tempStack
			if stack == nil {
				log.Fatal("Operator has one operand")
			}
			result := term.Operator.Operate(value1, value2)
			stack = Push(stack, result)
		} else {
			stack = Push(stack, term.Value)
		}
	}
	stack, result := Pop(stack)
	return result
}

func (expression *Expression) Add(terms ...Term) {
	for _, term := range terms {
		expression.Terms = Push(expression.Terms, term)
	}
}

func Precedence(operator Operator) (precedence int8) {
	switch operator {
	case "(":
		precedence = 0
	case "+", "-":
		precedence = 1
	case "*", "/":
		precedence = 2
	case "^":
		precedence = 3
	default:
		log.Fatal(operator, "has not attribute precedence")
	}
	return
}

func Peek[T comparable](stack []T) T {
	var t T
	if len(stack) == 0 {
		return t
	}
	return stack[len(stack)-1]
}

func Push[T comparable](stack []T, element T) []T {
	return append(stack, element)
}

func Pop[T comparable](stack []T) ([]T, T) {
	if len(stack) == 0 {
		var t T
		return nil, t
	}
	last := len(stack) - 1
	return stack[:last], stack[last]
}

func Update(stack OperatorStack, operator Operator) (OperatorStack, []Term) {
	var poppedOperators []Term
	if len(stack) == 0 || operator == "(" || Peek(stack) == "(" {
		return Push(stack, operator), poppedOperators
	}
	if ")" == operator {
		for len(stack) > 0 {
			tempStack, topOfStack := Pop(stack)
			stack = tempStack
			if "(" == topOfStack {
				return stack, poppedOperators
			} else {
				poppedOperators = append(poppedOperators, Term{Operator: topOfStack, IsOperator: true})
			}
		}
		log.Fatal("Right parenthesis has no matching left parenthesis")
	}
	if Precedence(operator) > Precedence(Peek(stack)) {
		return Push(stack, operator), poppedOperators
	}
	for len(stack) > 0 {
		topOfStack := Peek(stack)
		if Precedence(topOfStack) < Precedence(operator) {
			return Push(stack, operator), poppedOperators
		} else if topOfStack == "(" {
			return Push(stack, operator), poppedOperators
		} else {
			stack, topOfStack = Pop(stack)
			poppedOperators = append(poppedOperators, Term{Operator: topOfStack, IsOperator: true})
		}
	}
	return Push(stack, operator), poppedOperators
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
				memory[Identifier(assignee)] = expression.Evaluate()
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
	} else if !expression.IsEmpty() {
		fmt.Println(expression.Evaluate())
	}
}

func makeExpression(text string) (Expression, error) {
	reader := strings.NewReader(text)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanWords)
	expression := Expression{}
	var stack OperatorStack
	for scanner.Scan() {
		if value, ok := isNumber(scanner.Text()); ok {
			expression.Add(Term{Value: value})
		} else if value, ok = memory[Identifier(scanner.Text())]; ok {
			expression.Add(Term{Value: value})
		} else if isIdentifier(scanner.Text()) {
			return Expression{}, errors.New(UNKNOWN)
		} else if operator, ok := isOperator(scanner.Text()); ok {
			tempStack, poppedOperators := Update(stack, operator)
			stack = tempStack
			expression.Add(poppedOperators...)
		} else {
			return Expression{}, errors.New(INVALID)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for len(stack) > 0 {
		tempStack, operator := Pop(stack)
		stack = tempStack
		expression.Add(Term{
			Operator:   operator,
			IsOperator: true,
		})
	}
	return expression, nil
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
