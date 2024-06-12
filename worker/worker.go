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
	// чисто в теории если приложение будет развиваться, мы можем вызывать Run несколько раз (специально или по ошибке) - я бы создавал ресурсы client ? и previousPrices явно в консрукторе воркеа
	client := &http.Client{} // используешь по сути дефолтный клиент - надо еще прописать таймаут - иначе если что-то не так, он у тебя может навсегда зависнуть на каком-то запросе

	for {
		select {
		case <-ctx.Done():
			return
		default:
			for _, symbol := range w.Symbols { // думаю лучше ставить селект внутри этого цикла - в твоем случае, когда мы отправим стоп сигнал, придется ждать когда все символы из списка отработают
				resp, err := client.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol))

				w.mu.Lock() // я бы вынес в IncReqCount метод - мало ли, где еще потом захотим увеличить счетчик
				w.RequestsCount++
				w.mu.Unlock()

				if err != nil { // ошибку лучше поближе к запросу писать, ты же счетчик можешь инкрементить и до запроса
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
