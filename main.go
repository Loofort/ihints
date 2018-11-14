package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Loofort/хhints/scrape"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	cmdScrape = kingpin.Command("scrape", "scrape itunes hints")
	priority  = cmdScrape.Flag("priority", "set minimum desired hint priority").Default("0").Short('p').Int16()
	cmdLeaf   = kingpin.Command("leaf", "extract hints leaves")
	leafFile  = cmdLeaf.Arg("file", "hints file path").Required().String()
	cmdSort   = kingpin.Command("sort", "sort hints by text and query")
	sortFile  = cmdSort.Arg("file", "hints file path").Required().String()
	cmdUniq   = kingpin.Command("uniq", "extract unique hints")
	uniqFile  = cmdUniq.Arg("file", "hints file path").Required().String()
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
		dir := "data/" + time.Now().Format("2006-01-02") + "/"
		err := os.MkdirAll(dir, 0755)
		check(err)

		fmt.Printf("scrape hint for priority %d to folder %s\n", *priority, dir)
		scrape(int16(*priority), dir)
	case "sort":
		//start := time.Now()
		tips, err := NewTipsFromFile(*sortFile)
		check(err)

		//t1 := time.Since(start)
		SortTips(tips)

		//t2 := time.Since(start)
		//fmt.Printf("read file=%v; including sort=%v", t1, t2)
		for _, tip := range tips {
			fmt.Print(tip)
		}

	case "leaf":
		tips, err := NewTipsFromFile(*leafFile)
		check(err)
		SortTips(tips)

		tip := tips[0]
		for _, t := range tips[1:] {
			if t.Text != tip.Text {
				fmt.Print(tip)
				tip = t
				continue
			}

			if strings.HasPrefix(t.Query, tip.Query) {
				tip = t
				continue
			}

			fmt.Print(tip)
			tip = t
		}

	case "uniq":
		tips, err := NewTipsFromFile(*uniqFile)
		check(err)
		SortTips(tips)

		tip := tips[0]
		for _, t := range tips[1:] {
			if t.Text != tip.Text {
				fmt.Print(tip)
				tip = t
				continue
			}

			if t.Priority > tip.Priority {
				//log.Printf("unexpected priority order: %s > %s", t, tip)
				tip = t
			}
		}
	}
}

const (
	hintsName = "hints.tsv"
)

func Scrape(hintsfile string, priority int16) {

	qs := scrape.Generate("")
	pipe, wait := scrape.NewMemPipe(qs)

	fh, err := os.OpenFile(hintsfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	check(err)
	storage := scrape.SafeWriter{Writer: fh}

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

func leafIndex(hintsFile string) {
	root, err := NewIndexFromFile(hintsFile)
	check(err)

	f, err := os.OpenFile(hintsFile, os.O_RDONLY, 0644)
	check(err)
	r := bufio.NewReader(f)

	printLeaf := func(node *IndexTree) {
		if node.IsLeaf() {

			_, err = f.Seek(node.offset, 0)
			check(err)
			r.Reset(f)

			for i := 0; true; i++ {
				line, err := r.ReadBytes('\n')
				if err == io.EOF {
					return
				}
				check(err)

				pices := bytes.SplitN(line, []byte{'\t'}, 3)
				if bytes.Compare(node.name, pices[1]) != 0 {
					if i == 0 {
						log.Printf("node.name %s ,pices[1] %s\n", node.name, pices[1])
					}
					return
				}

				fmt.Printf("%s", line)
			}

		}
	}

	WalkTree(root, printLeaf)
}
