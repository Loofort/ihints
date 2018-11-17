package scrape

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/Loofort/хhints/hints"
)

const (
	maxUint16 = ^uint16(0)
	maxInt16  = int16(maxUint16 >> 1)

	NoHints      = -maxInt16 - 1
	ZeroPriority = int16(-51)

	letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789." // "-_" are ignored
)

type Pipe interface {
	Pull() (query string, done func())
	Push(queries []string)
}

// return true when no more query to scrape
func Iterate(pipe Pipe, storage io.Writer, priority int16) (bool, error) {
	// get new query to proccess
	q, done := pipe.Pull()
	if done == nil {
		return true, nil
	}
	defer done()

	// scrape hints from itunes
	hs, err := hints.Scrape(q, http.DefaultClient)
	if err != nil {
		return false, fmt.Errorf("can't scrape '%s': %v", q, err)
	}

	// save hs
	if len(hs) > 0 {
		storage.Write(hints.ToBytes(hs))
	}

	// generate new queries
	mark, err := Analize(hs)
	if err != nil {
		return false, fmt.Errorf("unexpected hints result for '%s': %v", q, err)
	}
	if mark >= priority || (priority == 0 && mark == ZeroPriority) {
		qs := Generate(q)
		pipe.Push(qs)
	}

	return false, nil
}

// returns max priority
// or negative count if len(hints) < 50
// also checks response assumptions, and return error if it's wrong.
// Priority starts from 0 (included) to over 10K
func Analize(hs []hints.Hint) (int16, error) {
	ln := len(hs)
	if ln == 0 {
		return NoHints, nil
	}
	if ln > 50 {
		return 0, fmt.Errorf("results count %d", ln)
	}

	p := hs[ln-1].Priority
	for _, h := range hs {
		if p > h.Priority {
			return 0, fmt.Errorf("Priority order error")
		}
	}

	if ln < 50 {
		return int16(-ln), nil
	}

	if p == 0 {
		p = ZeroPriority
	}

	return p, nil
}

func Generate(q string) []string {
	qs := make([]string, 0, len(letterBytes)+1)
	for _, b := range letterBytes {
		gen := q + string(b)
		qs = append(qs, gen)
	}

	if len(q) > 0 && q[len(q)-1] != ' ' {
		qs = append(qs, q+" ")
	}
	return qs
}

type SafeWriter struct {
	io.WriteCloser
	sync.Mutex
}

func (sf *SafeWriter) Write(p []byte) (int, error) {
	sf.Lock()
	defer sf.Unlock()
	return sf.WriteCloser.Write(p)
}
