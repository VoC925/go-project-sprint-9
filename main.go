package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	// 1. Функция Generator
	var num int64 = 1

	for {
		select {
		case <-ctx.Done():
			close(ch)
			return
		default:
			ch <- num
			fn(num)
			num++
		}
	}
}

func Worker(in <-chan int64, out chan<- int64) {
	// 2. Функция Worker
	for v := range in {
		out <- v
		time.Sleep(time.Millisecond * 1)
	}
	close(out)
}

func main() {
	chIn := make(chan int64)

	// 3. Создание контекста
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	var inputSum int64
	var inputCount int64

	go Generator(ctx, chIn, func(i int64) {
		atomic.AddInt64(&inputSum, i)
		atomic.AddInt64(&inputCount, 1)
	})

	const NumOut = 5
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	amounts := make([]int64, NumOut)
	chOut := make(chan int64, NumOut)

	var (
		wg sync.WaitGroup
	)

	// 4. Собираем числа из каналов outs
	for j := 0; j < NumOut; j++ {
		wg.Add(1)
		go func(numCh int) {
			defer wg.Done()
			for v := range outs[numCh] {
				chOut <- v
				amounts[numCh]++
			}
		}(j)
	}

	go func() {
		wg.Wait()
		close(chOut)
	}()

	var count int64
	var sum int64

	// 5. Читаем числа из результирующего канала
	for v := range chOut {
		count++
		sum += v
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}
