package main

import (
	"fmt"
	"time"
)

type Data struct {
	err   error
	val   string
	Child *Data
}

func sendData(mi int, c chan Data) {
	if mi == 0 {
		time.Sleep(3 * time.Second)
	}
	// time.Sleep(time.Duration(1-mi) * time.Second)
	// d := new(Data)
	// d.val = fmt.Sprintf("hey: %v", i)
	for i := 0; i < 3; i++ {
		c <- Data{val: fmt.Sprintf("mi=%v, i=%v", mi, i)}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	c := make(chan Data)
	go sendData(0, c)
	go sendData(1, c)

	for i := 0; i < 6; i++ {
		d := <-c
		fmt.Printf("Got: %+v\n", d)
	}
}
