package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Service started! Start sleep for 10 seconds")
	time.Sleep(10 * time.Second)
	fmt.Println("Sleep end")

}
