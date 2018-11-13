package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789." // "-_" are ignored

type offsetBytes struct {
	offset int64
	bytes  []byte
}

type Storage struct {
	minPriority int16
	hintsc      chan []byte
	progressc   chan offsetBytes
	idx         int64
	rmux        *sync.Mutex
	fr          *os.File
	rr          *bufio.Reader
	wmux        *sync.Mutex
	fw          *os.File
	wg          *sync.WaitGroup
	wgg         *sync.WaitGroup
	cmux        *sync.Mutex
	closec      chan struct{}
}

// produses storage that panics if faces any error during run time
func NewStorageNE(minPriority int16, queryFile, hintsFile string) (*Storage, func()) {
	wgg := new(sync.WaitGroup)
	// run hints writer
	// bunch of hints should be holistic so we protect it from concurrent write.
	hintsc := make(chan []byte)
	fh, err := os.OpenFile(hintsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	mustNE(err)
	wgg.Add(1)
	go func() {
		defer wgg.Done()
		defer fh.Close()
		for b := range hintsc {
			_, err := fh.Write(b)
			mustNE(err)
		}
	}()

	// run query progress recorder
	queryIndexFile := queryFile + ".idx"
	progressc := make(chan offsetBytes)
	fi, err := os.OpenFile(queryIndexFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	mustNE(err)
	wgg.Add(1)
	go func() {
		defer wgg.Done()
		defer fi.Close()
		for p := range progressc {
			_, err := fi.Seek(p.offset, 0)
			mustNE(err)
			_, err = fi.Write(p.bytes)
			mustNE(err)
		}
	}()

	// query reader
	fr, err := os.OpenFile(queryFile, os.O_RDONLY, 0644)
	mustNE(err)
	rr := bufio.NewReader(fr)

	// query writer
	fw, err := os.OpenFile(queryFile, os.O_APPEND|os.O_WRONLY, 0644)
	mustNE(err)

	stg := &Storage{
		minPriority: minPriority,
		hintsc:      hintsc,
		progressc:   progressc,
		rmux:        new(sync.Mutex),
		fr:          fr,
		rr:          rr,
		wmux:        new(sync.Mutex),
		fw:          fw,
		wg:          new(sync.WaitGroup),
		wgg:         wgg,
		cmux:        new(sync.Mutex),
		closec:      make(chan struct{}),
	}
	wait := func() { wgg.Wait() }
	return stg, wait
}

func (stg *Storage) Stop() {
	stg.cmux.Lock()
	defer stg.cmux.Unlock()

	select {
	case <-stg.closec:
		return
	default:
	}

	close(stg.closec)
	stg.wg.Wait()

	close(stg.hintsc)
	close(stg.progressc)
	stg.fr.Close()
	stg.fw.Close()
	stg.wgg.Wait()
}

// Returns query and it's query index.
// also returns done func that should be called after store the results.
// done indicates that the query processing is finished.
// done == nil if no more queries to process.
func (stg *Storage) Get() (string, int64, func()) {
	stg.rmux.Lock()
	defer stg.rmux.Unlock()

	select {
	case <-stg.closec:
		return "", 0, nil
	default:
	}

	q, err := stg.readQuery()
	if err == io.EOF {
		stg.wg.Wait()
		q, err = stg.readQuery()
		if err == io.EOF {
			stg.Stop()
			return "", 0, nil
		}
	}
	mustNE(err)

	stg.idx++
	stg.wg.Add(1)
	done := func() { stg.wg.Done() }
	return q, stg.idx, done
}

func (stg *Storage) readQuery() (string, error) {
	line, isPrefix, err := stg.rr.ReadLine()
	if err != nil {
		return "", err
	}
	if isPrefix {
		return "", fmt.Errorf("line prefix, query=%s", line)
	}

	return string(line), nil
}

// Saves results and progress
// if hints contains unexpected data it returns error
func (stg *Storage) Set(q string, idx int64, hints []Hint) error {
	// save hints
	if len(hints) > 0 {
		stg.hintsc <- hints2bytes(q, hints)
	}

	// save progress
	m, err := progressMark(hints)
	if err != nil {
		return err
	}
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(m))
	stg.progressc <- offsetBytes{idx * 2, b}

	// generate new queries
	if m >= stg.minPriority || (stg.minPriority == 0 && m == zeroPriority) {
		qs := generateQueries(q)
		stg.writeQueries(qs)
	}

	return nil
}

func (stg *Storage) writeQueries(qs []string) {
	stg.wmux.Lock()
	defer stg.wmux.Unlock()

	b := queries2bytes(qs)
	_, err := stg.fw.Write(b)
	mustNE(err)
}

func queries2bytes(qs []string) []byte {
	b := new(bytes.Buffer)
	for _, q := range qs {
		b.WriteString(q)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

func hints2bytes(q string, hints []Hint) []byte {
	b := new(bytes.Buffer)
	for _, hint := range hints {
		fmt.Fprintf(b, "%d\t%s\t%s\n", hint.Priority, q, hint.Text) // can't be error
	}
	return b.Bytes()
}

func mustNE(err error) {
	if err != nil {
		panic(err)
	}
}

/******************************* Algorithm Part ****************************************/

func generateQueries(q string) []string {
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

const (
	maxUint16    = ^uint16(0)
	maxInt16     = int16(maxUint16 >> 1)
	noHints      = -maxInt16 - 1
	zeroPriority = int16(-51)
)

// returns max priority
// or negative count if len(hints) < 50
// also checks response assumptions, and return error if it's wrong.
// Priority starts from 0 (included) to over 10K
func progressMark(hints []Hint) (int16, error) {
	ln := len(hints)
	if ln == 0 {
		return noHints, nil
	}
	if ln > 50 {
		return 0, fmt.Errorf("results count %d", ln)
	}

	p := hints[ln-1].Priority
	for _, h := range hints {
		if p > h.Priority {
			return 0, fmt.Errorf("Priority order error")
		}
	}

	if ln < 50 {
		return int16(-ln), nil
	}

	if p == 0 {
		p = zeroPriority
	}

	return p, nil
}
