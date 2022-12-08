package hn

import (
	"testing"
)

func isFilterOut(item Item) bool {
	if item.Type != "story" || item.Url == "" {
		return true
	}
	return false
}

func TestListStoryDetails(t *testing.T) {
	items, err := ListStoryDetails(2, isFilterOut)
	if err != nil {
		t.Error(err)
	}

	if len(items) != 2 {
		t.Errorf("%s\n", "items num error")
	}

	for _, item := range items {
		if isFilterOut(item) {
			t.Errorf("%s\n", "items contains filterd out item")
		}
	}
}
