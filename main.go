package main

import (
	"context"
	"fmt"
	"github.com/davidgodness/qhn/hn"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
)

type CustomItem struct {
	hn.Item
	Domain string `json:"domain"`
}

func isFilterOut(item hn.Item) bool {
	if item.Type != "story" || item.Url == "" {
		return true
	}
	return false
}

func parseDomain(urlStr string) string {
	if parsedUrl, err := url.Parse(urlStr); err == nil {
		hostSplit := strings.Split(parsedUrl.Host, ".")
		if hostSplit[0] == "www" {
			hostSplit = hostSplit[1:]
		}
		return strings.Join(hostSplit, ".")
	}

	return ""
}

func news(writer http.ResponseWriter, request *http.Request) {
	var items []CustomItem
	ret, err := hn.ListStoryDetails(30, isFilterOut)
	if err != nil {
		return
	}
	for _, item := range ret {
		items = append(items, CustomItem{Item: item, Domain: parseDomain(item.Url)})
	}
	tpl := template.Must(template.ParseFiles("templates/news.gohtml"))
	tpl.Execute(writer, items)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/news", news)

	srv := &http.Server{Addr: ":8080", Handler: mux}
	isIdleConnClosed := make(chan struct{})

	go func() {
		inter := make(chan os.Signal, 1)
		signal.Notify(inter, os.Interrupt)

		select {
		case <-inter:
			fmt.Println()
			log.Println("receive the interrupt signal")
		}
		log.Println("shutdown the server")
		err := srv.Shutdown(context.Background())
		if err != nil {
			log.Println(err)
		}
		close(isIdleConnClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Println(err)
	}

	<-isIdleConnClosed
}
