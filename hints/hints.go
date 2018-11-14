package hints

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
)

type Hint struct {
	Priority int16
	Query    string
	Text     string
}

func (hint Hint) String() string {
	priority := strconv.Itoa(int(hint.Priority))
	return priority + "\t" + hint.Query + "\t" + hint.Text
}

func ToBytes(hints []Hint) []byte {
	b := new(bytes.Buffer)
	for _, hint := range hints {
		fmt.Fprintf(b, "%s\n", hint) // can't be error
	}
	return b.Bytes()
}

func Sort(hs []Hint) {
	sort.Slice(hs, func(i, j int) bool {
		if hs[i].Text == hs[j].Text {
			return hs[i].Query < hs[j].Query
		}
		return hs[i].Text < hs[j].Text
	})
}

func FromFile(hintsFile string) ([]Hint, error) {
	f, err := os.OpenFile(hintsFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)

	hs := []Hint{}
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

		hint := Hint{
			Priority: int16(priority),
			Query:    string(pices[1]),
			Text:     string(pices[2][:len(pices[2])-1]), // cutoff last \n symbol
		}
		hs = append(hs, hint)
	}

	return hs, nil
}
