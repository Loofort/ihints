package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Loofort/xscrape/iostuff"
	"github.com/Loofort/xscrape/search/scrape"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	scrapeCmd    = kingpin.Command("scrape", "scrape itunes hints")
	scrapeInput  = scrapeCmd.Flag("input", "term file").Default("").Short('i').String()
	scrapeOutput = scrapeCmd.Flag("output", "hint file to write results").Default("").Short('o').String()
)

func check(err error) {
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func main() {
	switch kingpin.Parse() {
	case "scrape":
		Scrape(*scrapeInput, *scrapeOutput)
	}
}

func Scrape(termfile, searchesfile string) {
	r, err := iostuff.InputReader(termfile)
	check(err)
	defer r.Close()
	pipe, wait := iostuff.NewStreamPipe(r)

	storage, err := iostuff.OutputWriter(searchesfile)
	check(err)
	defer storage.Close()

	for i := 0; i < 1; i++ {
		go func() {
			var err error
			finish := false
			sleep := time.Minute / 20
			for !finish {
				start := time.Now()
				finish, err = scrape.Iterate(http.DefaultClient, pipe, storage, "")
				if err != nil {
					log.Printf("%v\n", err)
				}
				took := time.Since(start)
				time.Sleep(sleep - took)
			}
		}()
	}

	wait()
}
