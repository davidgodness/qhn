package hn

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Item struct {
	By          string   `json:"by"`
	Descendants int      `json:"descendants"`
	Id          uint64   `json:"id"`
	Kids        []uint64 `json:"kids"`
	Score       int      `json:"score"`
	Time        int      `json:"time"`
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Url         string   `json:"url"`
}

type FilterFuc func(item Item) bool

var mu = &sync.Mutex{}
var cache = make(map[uint64]Item)

func TopStories() ([]uint64, error) {
	var ret []uint64

	rsp, err := http.Get("https://hacker-news.firebaseio.com/v0/topstories.json?print=pretty")
	if err != nil {
		return nil, err
	}
	log.Println("GET", "https://hacker-news.firebaseio.com/v0/topstories.json?print=pretty")

	defer rsp.Body.Close()

	err = json.NewDecoder(rsp.Body).Decode(&ret)
	if err != nil {
		return nil, err
	}

	return ret, err
}

func QueryItem(itemId uint64, item chan<- Item) {
	if cachedItem, ok := cache[itemId]; ok {
		item <- cachedItem
		return
	}
	urlStr := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json?print=pretty", itemId)
	rsp, err := http.Get(urlStr)
	defer close(item)
	if err != nil {
		return
	}
	log.Println("GET", urlStr)
	ret := Item{}
	err = json.NewDecoder(rsp.Body).Decode(&ret)
	if err != nil {
		return
	}
	item <- ret
	mu.Lock()
	cache[itemId] = ret
	mu.Unlock()
}

func ListStoryDetails(num int, isFilterOut FilterFuc) ([]Item, error) {
	topItemIds, err := TopStories()
	if err != nil {
		return nil, err
	}

	if len(topItemIds) < num {
		num = len(topItemIds)
	}

	items := make([]Item, 0, num)

	offset := 0
	for len(items) < num {
		queryNum := num - len(items) + 2
		chs := make([]chan Item, 0, queryNum)
		for i := 0; i < queryNum; i++ {
			chs = append(chs, make(chan Item, 1))
		}
		for i, ch := range chs {
			go QueryItem(topItemIds[i+offset], ch)
		}
		for _, ch := range chs {
			item := <-ch
			if !isFilterOut(item) && len(items) < num {
				items = append(items, item)
			}
		}
		offset += queryNum
	}

	return items, nil
}
