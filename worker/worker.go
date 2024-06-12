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
	client         *http.Client      // Исправлено: добавление клиента в структуру Worker
	previousPrices map[string]string // Исправлено: добавление карты для предыдущих цен в структуру Worker
}

func NewWorker(symbols []string) *Worker {
	return &Worker{
		Symbols:        symbols,
		client:         &http.Client{Timeout: 10 * time.Second}, // Исправлено: установка таймаута для клиента
		previousPrices: make(map[string]string),                 // Исправлено: инициализация карты предыдущих цен
	}
}

func (w *Worker) Run(ctx context.Context, wg *sync.WaitGroup, priceChan chan<- models.PriceUpdate) {
	defer wg.Done()

	for {
		for _, symbol := range w.Symbols {
			select {
			case <-ctx.Done(): // Исправлено: селект для проверки ctx внутри цикла символов
				fmt.Println("Stopping Run goroutine...")
				return
			default:
				w.IncReqCount() // Исправлено: вызов метода для инкрементации счетчика запросов
				resp, err := w.client.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol))
				if err != nil { // Исправлено: перемещение логирования ошибок ближе к запросу
					log.Println("fetching price:", err)
					continue
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Println("reading response body:", err)
					continue
				}

				var ticker models.Ticker
				if err := json.Unmarshal(body, &ticker); err != nil {
					log.Println("unmarshalling JSON:", err)
					continue
				}
				priceChan <- models.PriceUpdate{Symbol: ticker.Symbol, Price: ticker.Price}
			}
		}
	}
}

func (w *Worker) IncReqCount() { // Исправлено: метод для инкрементации счетчика запросов
	w.mu.Lock()
	w.RequestsCount++
	w.mu.Unlock()
}

func (w *Worker) GetRequestsCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.RequestsCount
}
