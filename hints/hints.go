package hints

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
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
	scanner := bufio.NewScanner(reader)
	hs := []Hint{}
	for scanner.Scan() {
		line := scanner.Text()

		pices := strings.SplitN(line, "\t", 3)
		if len(pices) != 3 {
			return nil, fmt.Errorf("incorrect line: %s", line)
		}

		priority, err := strconv.Atoi(pices[0])
		if err != nil {
			return nil, fmt.Errorf("bad priority: %v", err)
		}

		hint := Hint{
			Priority: int16(priority),
			Query:    pices[1],
			Term:     pices[2],
		}
		hs = append(hs, hint)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hs, nil
}
