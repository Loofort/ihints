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

func main() {

	switch kingpin.Parse() {
	case "scrape":
		dir := "data/" + time.Now().Format("2006-01-02") + "/"
		err := os.MkdirAll(dir, 0755)
		mustNE(err)

		fmt.Printf("scrape hint for priority %d to folder %s\n", *priority, dir)
		scrape(int16(*priority), dir)
	case "sort":
		//start := time.Now()
		tips, err := NewTipsFromFile(*sortFile)
		mustNE(err)

		//t1 := time.Since(start)
		SortTips(tips)

		//t2 := time.Since(start)
		//fmt.Printf("read file=%v; including sort=%v", t1, t2)
		for _, tip := range tips {
			fmt.Print(tip)
		}

	case "leaf":
		tips, err := NewTipsFromFile(*leafFile)
		mustNE(err)
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
		mustNE(err)
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

func leafIndex(hintsFile string) {
	root, err := NewIndexFromFile(hintsFile)
	mustNE(err)

	f, err := os.OpenFile(hintsFile, os.O_RDONLY, 0644)
	mustNE(err)
	r := bufio.NewReader(f)

	printLeaf := func(node *IndexTree) {
		if node.IsLeaf() {

			_, err = f.Seek(node.offset, 0)
			mustNE(err)
			r.Reset(f)

			for i := 0; true; i++ {
				line, err := r.ReadBytes('\n')
				if err == io.EOF {
					return
				}
				mustNE(err)

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
