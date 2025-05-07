package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/talx-hub/gopher-bonus/internal/agent"
	"github.com/talx-hub/gopher-bonus/internal/model"
)

func main() {
	inputCh := make(chan uint64)
	outputCh := make(chan model.DTOAccrualInfo)
	a := agent.New(inputCh, outputCh, "127.0.0.1:8080")
	var wg sync.WaitGroup
	const nGoroutines = 3
	wg.Add(nGoroutines)

	go a.Run(context.TODO(), model.DefaultRequestCount)

	var n uint64
	go func() {
		for {
			_, err := fmt.Fscan(os.Stdin, &n)
			if err != nil {
				log.Fatal(err)
			}
			inputCh <- n
		}
	}()

	go func() {
		for {
			fmt.Println(<-outputCh)
		}
	}()

	wg.Wait()
}
