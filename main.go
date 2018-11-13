package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	priority  = kingpin.Flag("priority", "set minimum desired hint priority").Default("1000").Short('p').Int16()
	cmdScrape = kingpin.Command("scrape", "scrape itunes hints")
	cmdLeaf   = kingpin.Command("leaf", "extract hints leaves")
	leafFile  = cmdLeaf.Arg("file", "hints file path").Required().String()
)

func main() {

	dir := "data/" + time.Now().Format("2006-01-02") + "/"
	err := os.MkdirAll(dir, 0755)
	mustNE(err)

	switch kingpin.Parse() {
	case "scrape":
		fmt.Printf("scrape hint for priority %d to folder %s\n", *priority, dir)
		scrape(int16(*priority), dir)
	case "leaf":
		root, err := NewIndexFromFile(*leafFile)
		mustNE(err)

		f, err := os.OpenFile(*leafFile, os.O_RDONLY, 0644)
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
}
