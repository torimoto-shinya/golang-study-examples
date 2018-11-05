package main

import (
	"fmt"
	"sync"
)

type Fetcher interface {
	// Fetch returns the body of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

type SafeCache struct {
	visited map[string]bool
	routine int
	mux     sync.Mutex
}

func (c *SafeCache) Visited(url string) {
	c.mux.Lock()
	c.visited[url] = true
	c.mux.Unlock()
}

func (c *SafeCache) IsVisited(url string) bool {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.visited[url]
}

func (c *SafeCache) AddRoutine() {
	c.mux.Lock()
	c.routine++
	c.mux.Unlock()
}

func (c *SafeCache) DoneRoutine() {
	c.mux.Lock()
	c.routine--
	c.mux.Unlock()
}

func (c *SafeCache) GetRoutineNum() int {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.routine
}

type FetchResult struct {
	url  string
	body string
	err  error
}

func crawl(url string, depth int, fetcher Fetcher, ch chan FetchResult, cache *SafeCache) {
	if depth <= 0 {
		return
	}
	chanCloseFunc := func(ch chan FetchResult, cache *SafeCache) {
		if n := cache.GetRoutineNum(); n <= 0 {
			close(ch)
		}
	}

	cache.Visited(url)
	body, urls, err := fetcher.Fetch(url)

	if err != nil {
		fmt.Println(err)
		cache.DoneRoutine()
		chanCloseFunc(ch, cache)
		return
	}

	// チャネルに送信
	var result FetchResult
	result.url = url
	result.body = body
	result.err = err
	ch <- result

	for _, u := range urls {
		if !cache.IsVisited(u) {
			cache.AddRoutine()
			go crawl(u, depth-1, fetcher, ch, cache)
		}
	}

	cache.DoneRoutine()
	chanCloseFunc(ch, cache)

	return
}

// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher) {
	// TODO: Fetch URLs in parallel.
	// TODO: Don't fetch the same URL twice.
	// This implementation doesn't do either:

	// 読み書きを保証しているキャッシュ
	cache := &SafeCache{visited: make(map[string]bool), routine: 0}

	// 結果を受け取るためのチャネル
	ch := make(chan FetchResult)

	cache.AddRoutine()
	go crawl(url, depth, fetcher, ch, cache)
	for r := range ch {
		fmt.Printf("found: %s %q\n", r.url, r.body)
	}
	return
}

func main() {
	Crawl("https://golang.org/", 4, fetcher)
}

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher is a populated fakeFetcher.
var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}
