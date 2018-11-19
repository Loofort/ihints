package hints

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
)

type Hint struct {
	Priority int16
	Query    string
	Term     string
}

func (hint Hint) String() string {
	priority := strconv.Itoa(int(hint.Priority))
	return priority + "\t" + hint.Query + "\t" + hint.Term
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
		if hs[i].Term == hs[j].Term {
			return hs[i].Query < hs[j].Query
		}
		return hs[i].Term < hs[j].Term
	})
}

func FromReader(reader io.Reader) ([]Hint, error) {
	r := bufio.NewReader(reader)

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
			Term:     string(pices[2][:len(pices[2])-1]), // cutoff last \n symbol
		}
		hs = append(hs, hint)
	}

	return hs, nil
}
