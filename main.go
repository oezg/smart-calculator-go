package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const HELP = "The program calculates the sum of numbers"

func main() {
	var scanner *bufio.Scanner
	scanner = bufio.NewScanner(os.Stdin)
	mainloop(scanner)
	fmt.Println("Bye!")
}

func mainloop(scanner *bufio.Scanner) {
	for scanner.Scan() {
		if "/exit" == scanner.Text() {
			return
		}
		if "/help" == scanner.Text() {
			fmt.Println(HELP)
			continue
		}
		reader := strings.NewReader(scanner.Text())
		intScanner := bufio.NewScanner(reader)
		intScanner.Split(ScanInts)
		var nums []string
		for intScanner.Scan() {
			nums = append(nums, intScanner.Text())
		}
		if err := intScanner.Err(); err != nil {
			fmt.Println(err)
		}
		if len(nums) > 0 {
			fmt.Println(addition(nums...))
		}
	}
}

func addition(nums ...string) (sum int) {
	for _, num := range nums {
		number, err := strconv.Atoi(num)
		if err != nil {
			log.Fatal(err)
		}
		sum += number
	}
	return
}

func ScanInts(data []byte, atEOF bool) (advance int, token []byte, err error) {
	advance, token, err = bufio.ScanWords(data, atEOF)
	if err == nil && token != nil {
		_, err = strconv.Atoi(string(token))
	}
	return advance, token, err
}
