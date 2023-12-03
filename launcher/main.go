package main

import (
	"distributed-algos/algos"
	"fmt"
	"time"
)

func main() {
	lp, err := algos.NewLamportPeer(8080)
	if err != nil {
		panic(err)
	}

	go lp.Start()

	time.Sleep(5 * time.Second)
	fmt.Println("Waiting for server to start...")
	defer lp.Stop()

	lc := algos.NewLamportClient(8080)
	err = lc.Send('e', "Hello Word!")
	if err != nil {
		panic(err)
	}
	data, err := lc.Receive()
	if err != nil {
		panic(err)
	}
	strData := string(data)
	fmt.Printf("Main has received: %s\n", strData)

	err = lc.Send('p', "ping")
	if err != nil {
		panic(err)
	}
	data, err = lc.Receive()
	if err != nil {
		panic(err)
	}
	strData = string(data)
	fmt.Printf("Main has received: %s", strData)

	err = lc.Close()
	if err != nil {
		panic(err)
	}

	err = lc.Send('p', "pinky hi")
	if err != nil {
		panic(err)
	}
	data, err = lc.Receive()
	if err != nil {
		panic(err)
	}
	strData = string(data)
	fmt.Printf("Main has received: %s", strData)

	err = lc.Close()
	if err != nil {
		panic(err)
	}
}
