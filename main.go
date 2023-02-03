package main

import (
	"fmt"
	"log"
)

func main() {
	var i, j int
	_, err := fmt.Scan(&i, &j)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(i + j)
}
