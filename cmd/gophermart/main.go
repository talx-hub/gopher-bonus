package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/talx-hub/gopher-bonus/internal/model"
	"github.com/talx-hub/gopher-bonus/internal/service"
)

func main() {
	inputCh := make(chan uint64)
	outputCh := make(chan model.DTOAccrualInfo)
	a := service.New(inputCh, outputCh, "127.0.0.1:8080")
	var wg sync.WaitGroup
	wg.Add(3)

	go a.Run(context.TODO(), 100500)

	var n uint64
	go func() {
		for {
			fmt.Fscan(os.Stdin, &n)
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
