package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Loofort/xscrape/hints"
	"github.com/Loofort/xscrape/hints/scrape"
	"github.com/Loofort/xscrape/iostuff"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	scrapeCmd      = kingpin.Command("scrape", "scrape itunes hints")
	scrapePriority = scrapeCmd.Flag("priority", "set minimum desired hint priority").Default("0").Short('p').Int16()
	scrapeQuery    = scrapeCmd.Flag("query", "query file").Default("").Short('q').String()
	scrapeOutput   = scrapeCmd.Flag("output", "hint file to write results").Default("").Short('o').String()

	uniqCmd  = kingpin.Command("uniq", "extract unique hints")
	uniqFile = uniqCmd.Arg("file", "hints file path").String()

	leafCmd  = kingpin.Command("leaf", "extract hints leaves")
	leafFile = leafCmd.Arg("file", "hints file path").String()

	termCmd      = kingpin.Command("terms", "produce terms from hints")
	termFile     = termCmd.Arg("file", "hints file path").String()
	termPriority = termCmd.Flag("priority", "set minimum desired hint priority").Default("0").Short('p').Int16()
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
		Scrape(*scrapeQuery, *scrapeOutput, *scrapePriority)
	case "uniq":
		Uniq(*uniqFile)
	case "leaf":
		Leaf(*leafFile)
	case "terms":
		Term(*termFile, *termPriority)
	}
}

func Scrape(queryfile, hintsfile string, priority int16) {
	pipe, wait := scrapePipe(queryfile)

	storage, err := iostuff.OutputWriter(hintsfile)
	check(err)
	defer storage.Close()

	for i := 0; i < 10; i++ {
		go func() {
			var err error
			finish := false
			for !finish {
				finish, err = scrape.Iterate(pipe, storage, priority)
				if err != nil {
					log.Printf("%v\n", err)
				}
			}
		}()
	}

	wait()
}

func scrapePipe(queryfile string) (iostuff.Pipe, func() error) {
	r, err := iostuff.InputReader(queryfile)
	check(err)

	if r != nil {
		return iostuff.NewMemReaderPipe(r)
	}

	qs := scrape.Generate("")
	return iostuff.NewMemPipe(qs)
}

func Term(hintsfile string, priority int16) {
	r, err := iostuff.InputReader(hintsfile)
	check(err)
	defer r.Close()

	hs, err := hints.FromReader(r)
	check(err)

	for _, hint := range hs {
		if hint.Priority >= priority {
			fmt.Println(hint.Term)
		}
	}
}

func Uniq(filename string) {
	hs := sortedHints(filename)
	hint := hs[0]
	for _, t := range hs[1:] {
		if t.Term != hint.Term {
			fmt.Println(hint)
			hint = t
			continue
		}

		if t.Priority > hint.Priority {
			hint = t
		}
	}
}

func Leaf(filename string) {
	hs := sortedHints(filename)
	hint := hs[0]
	for _, t := range hs[1:] {
		if t.Term != hint.Term {
			fmt.Println(hint)
			hint = t
			continue
		}

		if strings.HasPrefix(t.Query, hint.Query) {
			hint = t
			continue
		}

		fmt.Println(hint)
		hint = t
	}
}

func sortedHints(hintsfile string) []hints.Hint {
	r, err := iostuff.InputReader(hintsfile)
	check(err)
	defer r.Close()

	hs, err := hints.FromReader(r)
	check(err)

	hints.Sort(hs)
	return hs
}
