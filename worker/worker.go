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
	"time"
)

type Worker struct {
	Symbols        []string
	RequestsCount  int
	mu             sync.Mutex
	client         *http.Client
	previousPrices map[string]string
}

func NewWorker(symbols []string) *Worker {
	return &Worker{
		Symbols:        symbols,
		client:         &http.Client{Timeout: 10 * time.Second},
		previousPrices: make(map[string]string),
	}
}

func (w *Worker) Run(ctx context.Context, wg *sync.WaitGroup, priceChan chan<- models.PriceUpdate) {
	defer wg.Done()
	for {
		for _, symbol := range w.Symbols {
			select {
			case <-ctx.Done():
				return
			default:
				w.IncReqCount()
				resp, err := w.client.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol))
				if err != nil {
					log.Println("fetching price:", err)
					continue
				}
				body, err := io.ReadAll(resp.Body)
				defer resp.Body.Close()
				if err != nil {
					log.Println("reading response body:", err)
					continue
				}

				var ticker models.Ticker
				if err := json.Unmarshal(body, &ticker); err != nil {
					log.Println("unmarshalling JSON:", err)
					continue
				}
				if ctx.Err() == nil {
					priceChan <- models.PriceUpdate{Symbol: ticker.Symbol, Price: ticker.Price}
				}
			}
		}
	}
}

func (w *Worker) IncReqCount() {
	w.mu.Lock()
	w.RequestsCount++
	w.mu.Unlock()
}

func (w *Worker) GetRequestsCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.RequestsCount
}
