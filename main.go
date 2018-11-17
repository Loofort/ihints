package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/Loofort/хhints/hints"
	"github.com/Loofort/хhints/scrape"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	scrapeCmd      = kingpin.Command("scrape", "scrape itunes hints")
	scrapePriority = scrapeCmd.Flag("priority", "set minimum desired hint priority").Default("0").Short('p').Int16()
	scrapeQuery    = scrapeCmd.Flag("query", "query file").Default("").Short('q').String()
	scrapeOutput   = scrapeCmd.Flag("output", "hint file to write results").Default("").Short('o').String()

	leafCmd  = kingpin.Command("leaf", "extract hints leaves")
	leafFile = leafCmd.Arg("file", "hints file path").Required().String()

	uniqCmd  = kingpin.Command("uniq", "extract unique hints")
	uniqFile = uniqCmd.Arg("file", "hints file path").Required().String()
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
	case "leaf":
		Leaf(*leafFile)
	case "uniq":
		Uniq(*uniqFile)
	}
}

func Scrape(queryfile, hintsfile string, priority int16) {
	qs, err := inputlines(queryfile)
	check(err)
	if len(qs) == 0 {
		qs = scrape.Generate("")
	}
	pipe, wait := scrape.NewMemPipe(qs)

	storage, err := outputWriter(hintsfile)
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

func Uniq(filename string) {
	hs := sortedHints(filename)
	hint := hs[0]
	for _, t := range hs[1:] {
		if t.Text != hint.Text {
			fmt.Print(hint)
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
		if t.Text != hint.Text {
			fmt.Print(hint)
			hint = t
			continue
		}

		if strings.HasPrefix(t.Query, hint.Query) {
			hint = t
			continue
		}

		fmt.Print(hint)
		hint = t
	}
}

func sortedHints(hintsfile string) []hints.Hint {
	r, err := inputReader(hintsfile)
	check(err)
	defer r.Close()

	hs, err := hints.FromReader(r)
	check(err)

	hints.Sort(hs)
	return hs
}

func inputlines(filename string) ([]string, error) {
	r, err := inputReader(filename)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	defer r.Close()

	lines, err := readlines(r)
	return lines, err
}

func readlines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func outputWriter(filename string) (io.WriteCloser, error) {
	if filename != "" {
		fh, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		return &scrape.SafeWriter{WriteCloser: fh}, nil
	}

	return &scrape.SafeWriter{WriteCloser: os.Stdout}, nil
}

func inputReader(filename string) (io.ReadCloser, error) {
	if filename != "" {
		file, err := os.Open(filename)
		return file, err
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Mode()&os.ModeCharDevice == 0 {
		return os.Stdin, nil
	}
	return nil, nil
}

/*
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
*/
