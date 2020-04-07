package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
)

/*Программа читает из stdin строки, содержащие URL.
На каждый URL нужно отправить HTTP-запрос методом GET
и посчитать кол-во вхождений строки "Go" в теле ответа.
В конце работы приложение выводит на экран общее количество
найденных строк "Go" во всех переданных URL, например:

$ echo -e 'https://golang.org\nhttps://golang.org' | go run 1.go
Count for https://golang.org: 9
Count for https://golang.org: 9
Total: 18

Каждый URL должен начать обрабатываться сразу после вычитывания
и параллельно с вычитыванием следующего.
URL должны обрабатываться параллельно, но не более k=5 одновременно.
Обработчики URL не должны порождать лишних горутин, т.е. если k=5,
а обрабатываемых URL-ов всего 2, не должно создаваться 5 горутин.

Нужно обойтись без глобальных переменных и использовать только стандартную библиотеку.*/

// Проверено линтером golangci lint

const substr = "go"

const countOfWorkers = 2

type Message struct {
	WG  *sync.WaitGroup
	URL string

	TotalCount *int
	MU         sync.RWMutex
}

/*
Я сделал две версии, так как перечитывая задание меня смутило вот это
"Обработчики URL не должны порождать лишних горутин, т.е. если k=5,
а обрабатываемых URL-ов всего 2, не должно создаваться 5 горутин."
Поэтому я сделал вторую версию без воркеров. Где горутина создается на каждый элемент. */

func main() {
	counter := make(chan int)
	ch := startWorkers(counter)
	sc := bufio.NewScanner(os.Stdin)

	wg := &sync.WaitGroup{}
	totalCount := 0

	go func() {
		for sc.Scan() {
			wg.Add(1)
			msg := &Message{
				WG:         wg,
				URL:        sc.Text(),
				TotalCount: &totalCount,
			}
			ch <- msg

		}
		wg.Wait()
		close(counter)
	}()
	count := 0
	for v := range counter {
		count += v
	}
	fmt.Println("Total by counter: ", count)
	fmt.Println("Total by mutex: ", totalCount)
}

func startWorkers(counter chan<- int) chan<- *Message {
	ch := make(chan *Message, 5)

	for i := 1; i <= countOfWorkers; i++ {
		go startWorker(ch, counter)
	}
	return ch
}

func startWorker(ch <-chan *Message, counter chan<- int) {
	for v := range ch {
		count := doWorkerWork(v)
		counter <- count
		v.MU.Lock()
		*v.TotalCount += count
		v.MU.Unlock()

		// У меня мысли, что мы сделаем done, вейтгрупа закончится, мы закроем counter и не успеем этот результ посчитать
		// поэтому переключаем ujрутину
		runtime.Gosched()
		v.WG.Done()
	}
}

func doWorkerWork(msg *Message) int {

	resp, err := http.Get(msg.URL)
	if err != nil {
		log.Println(err)
		return 0
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return 0
	}
	count := strings.Count(string(body), substr)

	fmt.Printf("Count for %s: %d \n", msg.URL, count)
	return count
}
