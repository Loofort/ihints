package iostuff

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

// Pipe implements the unlimited thread safe string FIFO
type Pipe interface {
	// Returns string and done func.
	// done indicates that the query processing is finished.
	// done == nil if no more queries to process.
	Pull() (task string, done func())

	// Adds new strings to the pipe
	Push(queries []string)

	// gracefully closes the pipe.
	// it waits for the all in-flight tasks to be finished.
	Close()
}

var Closed = fmt.Errorf("the pipe was closed")

type basePipe struct {
	wg      *sync.WaitGroup
	closec  chan struct{}
	taskc   chan string
	resultc chan []string
}

// creates and return memory pipe
// also return wait func that returns when pipe is done
func newBasePipe(taskc chan string, resultc chan []string) (basePipe, func() error) {
	pipe := basePipe{
		wg:      new(sync.WaitGroup),
		closec:  make(chan struct{}),
		taskc:   taskc,
		resultc: resultc,
	}

	pipe.wg.Add(1)

	wait := func() error {
		pipe.wg.Done()
		pipe.wg.Wait()
		close(pipe.taskc)
		select {
		case <-pipe.closec:
			return Closed
		default:
			return nil
		}
	}
	return pipe, wait
}

func (pipe basePipe) Close() {
	close(pipe.closec)
}

func (pipe basePipe) Pull() (string, func()) {
	query, ok := <-pipe.taskc
	if !ok {
		return "", nil
	}

	pipe.wg.Add(1)
	done := func() { pipe.wg.Done() }
	return query, done
}

func (pipe basePipe) Push(queries []string) {
	pipe.resultc <- queries
}

/******************* Pipe implementations **********************/

func NewMemPipe(queries []string) (Pipe, func() error) {
	pipe, wait := newBasePipe(make(chan string), make(chan []string))
	go memLoop(pipe, queries)
	return pipe, wait
}

func memLoop(pipe basePipe, queries []string) {
	query := ""
	taskc := pipe.taskc
	for {

		if len(queries) == 0 {
			if taskc != nil {
				taskc = nil
				pipe.wg.Done()
			}
		} else {
			if taskc == nil {
				taskc = pipe.taskc
				pipe.wg.Add(1)
			}

			if query == "" {
				query = queries[0]
				queries = queries[1:]
			}
		}

		select {
		case taskc <- query:
			query = ""
		case qs := <-pipe.resultc:
			queries = append(queries, qs...)
		case <-pipe.closec:
			if taskc != nil {
				pipe.wg.Done()
			}
			return
		}
	}
}

func NewReaderPipe(r io.Reader) (Pipe, func() error) {
	pipe, wait := newBasePipe(make(chan string), nil)

	scanner := bufio.NewScanner(r)
	pipe.wg.Add(1)
	go readerLoop(pipe, scanner)

	waitScn := func() error {
		wait()
		return scanner.Err()
	}
	return pipe, waitScn
}

func readerLoop(pipe basePipe, scanner *bufio.Scanner) {
	defer pipe.wg.Done()
	for {
		if !scanner.Scan() {
			return
		}

		line := scanner.Text()
		select {
		case pipe.taskc <- line:
		case <-pipe.closec:
			return
		}
	}
}

type memReaderPipe struct {
	read    Pipe
	mem     Pipe
	current Pipe
	rwmux   *sync.RWMutex
}

func NewMemReaderPipe(r io.Reader) (Pipe, func() error) {
	rpipe, rwait := NewReaderPipe(r)
	mpipe, mwait := NewMemPipe(nil)

	pipe := memReaderPipe{
		read:    rpipe,
		mem:     mpipe,
		current: rpipe,
		rwmux:   new(sync.RWMutex),
	}

	wait := func() error {
		err := rwait()
		if err != nil {
			mpipe.Close()
			return err
		}

		pipe.rwmux.Lock()
		pipe.current = mpipe
		pipe.rwmux.Unlock()

		return mwait()
	}
	return pipe, wait
}

func (pipe memReaderPipe) Pull() (task string, done func()) {
	pipe.rwmux.RLock()
	defer pipe.rwmux.RUnlock()
	return pipe.current.Pull()
}

func (pipe memReaderPipe) Push(queries []string) {
	pipe.mem.Push(queries)
}

func (pipe memReaderPipe) Close() {
	pipe.current.Close()
}
