package main

import (
	"Ticker-Parser/models"
	"Ticker-Parser/worker"
	"bufio"
	"context"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

func main() {
	configPath := "config.yaml"
	file, err := os.Open(configPath)
	if err != nil {
		log.Fatalf("Opening config file: %v", err)
	}
	defer file.Close()

	var config models.Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Decoding config file: %v", err)
	}

	numCPU := runtime.NumCPU()
	if config.MaxWorkers > numCPU {
		config.MaxWorkers = numCPU
	}

	if config.MaxWorkers > len(config.Symbols) {
		config.MaxWorkers = len(config.Symbols)
	}

	workers := make([]*worker.Worker, config.MaxWorkers)
	for i := 0; i < config.MaxWorkers; i++ {
		workers[i] = worker.NewWorker([]string{})
	}

	for i, symbol := range config.Symbols {
		workers[i%config.MaxWorkers].Symbols = append(workers[i%config.MaxWorkers].Symbols, symbol)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	priceChan := make(chan models.PriceUpdate, 100)
	previousPrices := make(map[string]string)
	var mu sync.Mutex

	for _, w := range workers {
		wg.Add(1)
		go w.Run(ctx, &wg, priceChan)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				// Прочитать оставшиеся элементы в канале
				for priceUpdate := range priceChan {
					mu.Lock()
					changed := ""
					if previousPrice, ok := previousPrices[priceUpdate.Symbol]; ok && previousPrice != priceUpdate.Price {
						changed = "changed"
					}
					previousPrices[priceUpdate.Symbol] = priceUpdate.Price
					mu.Unlock()
					fmt.Printf("%s price:%s%s\n", priceUpdate.Symbol, priceUpdate.Price, changed)
				}
				close(priceChan)
				return
			case priceUpdate := <-priceChan:
				mu.Lock()
				changed := ""
				if previousPrice, ok := previousPrices[priceUpdate.Symbol]; ok && previousPrice != priceUpdate.Price {
					changed = "changed"
				}
				previousPrices[priceUpdate.Symbol] = priceUpdate.Price
				mu.Unlock()
				fmt.Printf("%s price:%s%s\n", priceUpdate.Symbol, priceUpdate.Price, changed)
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				totalRequests := 0
				for _, wrk := range workers {
					totalRequests += wrk.GetRequestsCount()
				}
				fmt.Printf("workers requests total: %d\n", totalRequests)
			}
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stopChan
		cancel()
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if scanner.Text() == "STOP" {
			cancel()
			break
		}
	}

	wg.Wait()
	fmt.Println("All workers stopped.")
}
