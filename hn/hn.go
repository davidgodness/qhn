package hn

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
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

const baseUrl = "https://hacker-news.firebaseio.com/v0"

type FilterFuc func(item Item) bool

var mu = &sync.Mutex{}
var cache = make(map[uint64]Item)

func TopStories() ([]uint64, error) {
	var ret []uint64

	rsp, err := http.Get(baseUrl + "/topstories.json?print=pretty")
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()

	err = json.NewDecoder(rsp.Body).Decode(&ret)
	if err != nil {
		return nil, err
	}

	return ret, err
}

func QueryItem(itemId uint64) (Item, error) {
	ret := Item{}
	urlStr := fmt.Sprintf(baseUrl+"/item/%d.json?print=pretty", itemId)
	rsp, err := http.Get(urlStr)
	if err != nil {
		return ret, err
	}
	log.Println("GET", urlStr)

	err = json.NewDecoder(rsp.Body).Decode(&ret)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

type job struct {
	id     int
	itemId uint64
}

type result struct {
	id int
	Item
	err error
}

func worker(jobs chan job, results chan result) {
	for job := range jobs {
		initResult := result{id: job.id}
		if cachedItem, ok := cache[job.itemId]; ok {
			initResult.Item = cachedItem
			results <- initResult
			continue
		}

		ret, err := QueryItem(job.itemId)
		if err != nil {
			initResult.err = err
		} else {
			initResult.Item = ret
		}
		mu.Lock()
		cache[job.itemId] = ret
		mu.Unlock()
		results <- initResult
	}
}

func QueryItems(itemIds []uint64) ([]Item, error) {

	jobs := make(chan job, len(itemIds))
	results := make(chan result, len(itemIds))

	// limit the gorouting number
	for i := 0; i < 4; i++ {
		go worker(jobs, results)
	}

	for i, id := range itemIds {
		jobs <- job{
			id:     i,
			itemId: id,
		}
	}
	close(jobs)

	items := make([]Item, 0, len(itemIds))
	resList := make([]result, 0, len(itemIds))

	for i := 0; i < len(itemIds); i++ {
		resList = append(resList, <-results)
	}

	sort.Slice(resList, func(i, j int) bool {
		return resList[i].id < resList[j].id
	})

	for _, res := range resList {
		items = append(items, res.Item)
	}

	return items, nil
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

		ret, err := QueryItems(topItemIds[offset : offset+queryNum])
		if err != nil {
			return nil, err
		}

		for _, item := range ret {
			if len(items) >= num {
				break
			}
			if !isFilterOut(item) {
				items = append(items, item)
			}
		}

		offset += queryNum
	}

	return items, nil
}
