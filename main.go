package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/atis/quiet_hn/hn"
)

var vmi []string = []string{"machine learning", "ml", "artificial intelligence", "ai",
	"deep learning", "dl", "neural network", "nn", "computer vision", "cv"}


func main() {
	// parse flags
	var port, numStories int
	flag.IntVar(&port, "port", 3000, "the port to start the web server on")
	flag.IntVar(&numStories, "num_stories", 30, "the number of top stories to display")
	flag.Parse()

	tpl := template.Must(template.ParseFiles("./index.gohtml"))

	http.HandleFunc("/", handler(numStories, tpl))

	// Start the server
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handler(numStories int, tpl *template.Template) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var client hn.Client
		ids, err := client.TopItems()
		if err != nil {
			http.Error(w, "Failed to load top stories", http.StatusInternalServerError)
			return
		}

		stories := getStories(ids, numStories, client)

		data := templateData{
			Stories: stories,
			Time:    time.Now().Sub(start),
		}
		err = tpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Failed to process the template", http.StatusInternalServerError)
			return
		}
	})
}

func getStories(ids []int, numStories int, client hn.Client) []item {
	var stories []item
	ch := make(chan hn.Result)

	for i, id := range ids {
		go client.GetItem(i, id, ch)

		//using waitgroups to check if more routines needed or need stopping?
		if i%30 == 0 {
			time.Sleep(time.Second * 3)
		}

		if len(stories) >= numStories || i > 40 {
			break
		}
		i++
	}
	for result := range ch {
		if result.Error != nil {
			continue
		}
		item := parseHNItem(result.Item)

		if isStoryLink(item) {
			stories = append(stories, item)
			if len(stories) >= numStories {
				break
			}
		}
	}
	return stories
}

func isStoryLink(item item) bool {
	return item.Type == "story" && item.URL != ""
}

func parseHNItem(hnItem hn.Item) item {
	ret := item{Item: hnItem}
	url, err := url.Parse(ret.URL)
	if err == nil {
		ret.Host = strings.TrimPrefix(url.Hostname(), "www.")
	}
	return ret
}

// item is the same as the hn.Item, but adds the Host field
type item struct {
	hn.Item
	Host string
}

type templateData struct {
	Stories []item
	Time    time.Duration
}
