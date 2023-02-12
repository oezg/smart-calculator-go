package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	UNKNOWN = "Unknown variable "
	INVALID = "Invalid "
	EMPTY   = "empty expression"
	HELP    = `Smart calculator commands:
/clear	clears all variables
/vars	prints variables
/del	deletes variables (space separated)
/con	converts infix to postfix notation
/read	reads given file and updates variables
/write	writes variables to given file
/help	prints help
/exit	exits program

Smart calculator operations:
(   )   +   -   *   /   %   ^

Smart calculator supports only Latin characters for variables
Smart calculator supports only integers for numerical types.`
)

var memory = make(map[Identifier]Value)

type (
	Identifier    string
	Value         int
	Operator      string
	ValueStack    []Value
	OperatorStack []Operator
	Expression    []Term
)

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
	fmt.Println("+-+-+   Welcome to Smart Calculator   */*/*")
	fmt.Println("Enter a command or start calculation or type '/help'")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		handleCommand(scanner.Text())
	}
}

func handleCommand(text string) {
	if !strings.HasPrefix(text, "/") {
		handleAssignment(text)
		return
	}
	commands := strings.SplitN(text[1:], " ", 2)
	command := commands[0]
	switch command {
	case "exit":
		fmt.Println("Bye!")
		os.Exit(0)
	case "help":
		fmt.Println(HELP)
	case "vars":
		printVariables(os.Stdout)
	case "clear":
		memory = make(map[Identifier]Value)
	case "con":
		if IsEmpty(commands[1:]) {
			return
		}
		convertExpression(commands[1])
	case "del":
		if IsEmpty(commands[1:]) {
			return
		}
		deleteVariables(commands[1])
	case "read":
		if IsEmpty(commands[1:]) {
			return
		}
		readVariables(commands[1])
	case "write":
		if IsEmpty(commands[1:]) {
			return
		}
		writeVariables(commands[1])
	default:
		fmt.Println("Unknown command")
	}
}

func handleAssignment(text string) {
	if !strings.Contains(text, "=") {
		handleExpression(text)
		return
	}
	assignmentSides := strings.SplitN(text, "=", 2)
	assignee := strings.TrimSpace(assignmentSides[0])
	assigned := strings.TrimSpace(assignmentSides[1])
	if !isIdentifier(assignee) {
		fmt.Println("Invalid identifier")
		return
	}
	if result, err := evaluateExpression(assigned); err == nil {
		memory[Identifier(assignee)] = result
	} else {
		must(err, "assignment")
	}
}

func handleExpression(text string) {
	text = strings.TrimSpace(text)
	if result, err := evaluateExpression(text); err == nil {
		fmt.Println(result)
	} else {
		must(err, "expression")
	}
}

func evaluateExpression(text string) (value Value, err error) {
	var expression Expression
	if expression, err = convert2Postfix(text); err != nil {
		return
	}
	value, err = expression.Evaluate()
	return
}

func convert2Postfix(text string) (expression Expression, err error) {
	if 0 == len(text) {
		err = errors.New(EMPTY)
		return
	}
	reader := strings.NewReader(text)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanRunes)
	var stack OperatorStack
	var currentTerm, lastTerm RawTerm
	var term Term
	for scanner.Scan() {
		currentTerm.Close(lastTerm, scanner.Text())
		if currentTerm.closed {
			if term, err = validate(currentTerm); err != nil {
				return
			}
			if err = expression.Grow(&stack, term); err != nil {
				return
			}
			lastTerm, currentTerm = currentTerm, RawTerm{}
		}
		if err = currentTerm.Extend(scanner.Text()); err != nil {
			return
		}
	}
	if term, err = validate(currentTerm); err != nil {
		return
	}
	if err = expression.Grow(&stack, term); err != nil {
		return
	}
	for !IsEmpty(stack) {
		var operator Operator
		stack, operator = Pop(stack)
		expression.Add(Term{Operator: operator, IsOperator: true})
	}
	return
}

func (expression *Expression) Evaluate() (value Value, err error) {
	if IsEmpty(*expression) {
		err = errors.New(EMPTY)
		return
	}
	var stack ValueStack
	var value1, value2, result Value
	for _, term := range *expression {
		if !term.IsOperator {
			stack = Push(stack, term.Value)
			continue
		}
		stack, value1 = Pop(stack)
		stack, value2 = Pop(stack)
		if stack == nil {
			err = errors.New(INVALID)
		}
		result = term.Operator.Operate(value2, value1)
		stack = Push(stack, result)
	}
	value = Peek(stack)
	return
}

func (expression *Expression) Add(terms ...Term) {
	for _, term := range terms {
		*expression = Push(*expression, term)
	}
}

func (expression *Expression) Grow(stack *OperatorStack, term Term) error {
	if term.IsOperator {
		poppedOperators, err := stack.Update(term.Operator)
		if err != nil {
			return err
		}
		expression.Add(poppedOperators...)
	} else {
		expression.Add(term)
	}
	return nil
}

func (expression *Expression) String() string {
	terms := make([]string, 0, len(*expression))
	for _, term := range *expression {
		if term.IsOperator {
			terms = Push(terms, string(term.Operator))
		} else {
			terms = Push(terms, strconv.Itoa(int(term.Value)))
		}
	}
	return strings.Join(terms, " ")
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
	}
	return
}

func validate(term RawTerm) (validated Term, err error) {
	if term.isOperator {
		if operator, ok := isOperator(term.Text); ok {
			validated = Term{Operator: operator, IsOperator: true}
		} else {
			err = errors.New(INVALID)
		}
	} else if term.isValue {
		if value, ok := isNumber(term.Text); ok {
			validated = Term{Value: value}
		} else {
			err = errors.New(INVALID)
		}
	} else if term.isIdentifier {
		if value, ok := memory[Identifier(term.Text)]; ok {
			validated = Term{Value: value}
		} else if isIdentifier(term.Text) {
			err = errors.New(UNKNOWN)
		} else {
			err = errors.New(INVALID)
		}
	}
	return
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

func must(err error, statement string) {
	if err.Error() != EMPTY {
		printError(err.Error(), statement)
	}
}

func printError(message, statement string) {
	if message == INVALID {
		message += statement
	}
	fmt.Println(message)
}

func sortVariables() (identifiers []Identifier) {
	identifiers = make([]Identifier, 0, len(memory))
	for identifier := range memory {
		identifiers = Push(identifiers, identifier)
	}
	sort.Slice(identifiers, func(i, j int) bool {
		return identifiers[i] < identifiers[j]
	})
	return
}

func printVariables(writer io.Writer) {
	for _, identifier := range sortVariables() {
		_, err := fmt.Fprintf(writer, "%s = %d\n", identifier, memory[identifier])
		if err != nil {
			fmt.Println(err)
		}
	}
}

func deleteVariables(text string) {
	arguments := strings.Split(text, " ")
	for _, variable := range arguments {
		delete(memory, Identifier(variable))
	}
}

func readVariables(text string) {
	file, err := os.Open(text)
	if err != nil {
		fmt.Println(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		handleAssignment(scanner.Text())
	}
}

func writeVariables(text string) {
	file, err := os.Create(text)
	if err != nil {
		fmt.Println(err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(file)
	printVariables(file)
}

func convertExpression(text string) {
	infixExpression := strings.TrimSpace(text)
	postfixExpression, err := convert2Postfix(infixExpression)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(postfixExpression.String())
}
