package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"sort"
	"sync"
	"time"
)

const (
	UrlTemplate           = "https://content.cdn.viber.com/apps/icons/100/%d/icon.png"
	UrlCount              = 20000
	ParallelRequestsCount = 100
)

type IconInfo struct {
	id        int
	url       string
	imageType string
	err       error
}

func doRequest(id int) (*IconInfo, error) {
	url := fmt.Sprintf(UrlTemplate, id)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Close = true

	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, nil
	}

	_, t, err := image.Decode(response.Body)
	if err != nil {
		return &IconInfo{id, url, t, err}, nil
	}

	if t != "png" {
		return &IconInfo{id, url, t, nil}, nil
	}

	return nil, nil
}

func process() []IconInfo {
	var wg sync.WaitGroup
	limit := make(chan bool, ParallelRequestsCount)
	iic := make(chan IconInfo)
	finish := make(chan bool)
	infos := []IconInfo{}

	go func() {
		for value := range iic {
			infos = append(infos, value)
		}
		finish <- true
	}()

	for i := 0; i < UrlCount; i++ {
		wg.Add(1)
		limit <- true
		go func(id int, c chan IconInfo) {
			defer wg.Done()
			info, _ := doRequest(id)
			if info != nil {
				c <- *info
			}
			<-limit
		}(i, iic)
	}

	wg.Wait()
	close(iic)
	<-finish

	return infos
}

func printResults(infos []IconInfo) {
	sort.Slice(infos, func(i, j int) bool { return infos[i].id < infos[j].id })
	for _, i := range infos {
		if i.err != nil {
			fmt.Printf("url: %s | type: %s | error: %s\n", i.url, i.imageType, i.err)
		} else {
			fmt.Printf("url: %s | type: %s\n", i.url, i.imageType)
		}
	}

	fmt.Printf("total count = %d\n", len(infos))
}

func main() {
	start := time.Now()

	results := process()
	printResults(results)

	fmt.Printf("ExecutionTime %s\n", time.Now().Sub(start))
}
