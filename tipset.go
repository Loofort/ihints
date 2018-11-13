package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
)

type Tip struct {
	Priority int16
	Query    string
	Text     string
}

func (tip Tip) String() string {
	priority := strconv.Itoa(int(tip.Priority))
	return priority + "\t" + tip.Query + "\t" + tip.Text
}

func NewTipsFromFile(hintsFile string) ([]Tip, error) {
	f, err := os.OpenFile(hintsFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)

	tipset := []Tip{}
	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		pices := bytes.SplitN(line, []byte{'\t'}, 3)
		if len(pices) != 3 {
			return nil, fmt.Errorf("incorrect line: %s", line)
		}

		priority, err := strconv.Atoi(string(pices[0]))
		if err != nil {
			return nil, fmt.Errorf("bad priority: %v", err)
		}

		tip := Tip{
			Priority: int16(priority),
			Query:    string(pices[1]),
			Text:     string(pices[2]),
		}
		tipset = append(tipset, tip)
	}

	return tipset, nil
}

func SortTips(tips []Tip) {
	sort.Slice(tips, func(i, j int) bool {
		if tips[i].Text == tips[j].Text {
			return tips[i].Query < tips[j].Query
		}
		return tips[i].Text < tips[j].Text
	})
}
