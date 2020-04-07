package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
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

const substr = "go"

const countOfGoroutines = 2

type Message struct {
	Counter           chan<- int
	CounterGoroutines <-chan int
	URL               string

	WG *sync.WaitGroup

	MU         sync.Mutex
	TotalCount *int
}

func main() {
	counterSubstr := make(chan int)
	counterGoroutines := make(chan int, countOfGoroutines)
	counterByAtomic := int32(0)
	sc := bufio.NewScanner(os.Stdin)

	wg := &sync.WaitGroup{}
	totalCount := 0

	go func() {
		for sc.Scan() {

			atomic.AddInt32(&counterByAtomic, 1)

			counterGoroutines <- 1
			wg.Add(1)

			msg := &Message{
				CounterGoroutines: counterGoroutines,
				Counter:           counterSubstr,
				WG:                wg,
				URL:               sc.Text(),
				TotalCount:        &totalCount,
			}
			go doWorkerWork(msg)

		}
		wg.Wait()
		close(counterSubstr)
	}()
	count := 0
	for v := range counterSubstr {
		count += v
	}
	fmt.Println("Total by counter: ", count)
	fmt.Println("Total by mutex: ", totalCount)
}

func doWorkerWork(msg *Message) {

	resp, err := http.Get(msg.URL)
	if err != nil {
		log.Println(err)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	count := strings.Count(string(body), substr)

	fmt.Printf("Count for %s: %d \n", msg.URL, count)
	msg.Counter <- count
	msg.MU.Lock()
	*msg.TotalCount += count
	msg.MU.Unlock()
	msg.WG.Done()
	<-msg.CounterGoroutines
}
