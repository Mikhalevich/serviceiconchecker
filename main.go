package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/Mikhalevich/pbw"
)

const (
	URLTemplate           = "https://content.cdn.viber.com/apps/icons/100/%d/icon.png"
	URLCount              = 20000
	ParallelRequestsCount = 100
)

type IconInfo struct {
	id        int
	url       string
	imageType string
	err       error
}

func doRequest(id int) (*IconInfo, error) {
	url := fmt.Sprintf(URLTemplate, id)
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

func process() ([]IconInfo, []error) {
	var wg sync.WaitGroup
	limit := make(chan bool, ParallelRequestsCount)
	iic := make(chan IconInfo)
	errc := make(chan error)
	finish := make(chan bool)
	finishError := make(chan bool)
	infos := []IconInfo{}
	errs := []error{}
	progressChan := make(chan int64, 1000)

	pbw.ShowWithMax(progressChan, URLCount)

	go func() {
		for value := range iic {
			infos = append(infos, value)
		}
		finish <- true
	}()

	go func() {
		for err := range errc {
			errs = append(errs, err)
		}
		finishError <- true
	}()

	for i := 0; i < URLCount; i++ {
		wg.Add(1)
		limit <- true
		go func(id int, c chan IconInfo, ec chan error) {
			defer func() {
				wg.Done()
				<-limit
				progressChan <- 1
			}()
			info, err := doRequest(id)

			if err != nil {
				ec <- err
			}

			if info != nil {
				c <- *info
			}
		}(i, iic, errc)
	}

	wg.Wait()
	close(iic)
	close(errc)
	<-finish
	<-finishError

	return infos, errs
}

func printErrors(errs []error) {
	fmt.Println("Errors:")
	for _, err := range errs {
		fmt.Println(err)
	}
}

func printResults(infos []IconInfo) {
	sort.Slice(infos, func(i, j int) bool { return infos[i].id < infos[j].id })

	fmt.Println("Results:")
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

	results, errs := process()
	printErrors(errs)
	printResults(results)

	fmt.Printf("ExecutionTime %s\n", time.Now().Sub(start))
}
