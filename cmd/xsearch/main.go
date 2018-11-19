package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Loofort/xscrape/iostuff"
	"github.com/Loofort/xscrape/searches/scrape"
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
	pipe, wait := iostuff.NewMemReaderPipe(r)

	storage, err := iostuff.OutputWriter(searchesfile)
	check(err)
	defer storage.Close()

	for i := 0; i < 10; i++ {
		go func() {
			var err error
			finish := false
			for !finish {
				finish, err = scrape.Iterate(http.DefaultClient, pipe, storage, "")
				if err != nil {
					log.Printf("%v\n", err)
				}
			}
		}()
	}

	wait()
}
