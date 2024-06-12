package worker

import (
	"Ticker-Parser/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

type Worker struct {
	Symbols       []string
	RequestsCount int
	mu            sync.Mutex
}

func (w *Worker) Run(ctx context.Context, wg *sync.WaitGroup, priceChan chan<- models.PriceUpdate) {
	defer wg.Done()
	client := &http.Client{}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			for _, symbol := range w.Symbols {
				resp, err := client.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol))

				w.mu.Lock()
				w.RequestsCount++
				w.mu.Unlock()

				if err != nil {
					log.Println("Error fetching price:", err)
					continue
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Println("Error reading response body:", err)
					continue
				}

				var ticker models.Ticker
				if err := json.Unmarshal(body, &ticker); err != nil {
					log.Println("Error unmarshalling JSON:", err)
					continue
				}

				priceChan <- models.PriceUpdate{Symbol: ticker.Symbol, Price: ticker.Price}
			}
		}
	}
}

func (w *Worker) GetRequestsCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.RequestsCount
}
