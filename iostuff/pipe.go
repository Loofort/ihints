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
	line, ok := <-pipe.taskc
	return pipe.pull(line, ok)
}

func (pipe basePipe) pull(line string, ok bool) (string, func()) {
	if !ok {
		return "", nil
	}

	pipe.wg.Add(1)
	done := func() { pipe.wg.Done() }
	return line, done
}

func (pipe basePipe) Push(queries []string) {
	pipe.resultc <- queries
}

/******************* Pipe implementations **********************/

func NewBufferPipe(queries []string) (basePipe, func() error) {
	pipe, wait := newBasePipe(make(chan string), make(chan []string))
	pipe.wg.Add(1)
	go bufferLoop(pipe, queries)
	return pipe, wait
}

func bufferLoop(pipe basePipe, queries []string) {
	defer pipe.wg.Done()
	query := ""
	taskc := pipe.taskc
	if len(queries) == 0 {
		pipe.wg.Add(1)
	}
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

func NewReaderPipe(r io.Reader) (basePipe, func() error) {
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
	mpipe, mwait := NewBufferPipe(nil)

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

/**************************** *****************************************/
// streamPipe reads from reader as a first priority or from buffer as a secondary
type streamPipe struct {
	reader basePipe
	buffer basePipe
}

func NewStreamPipe(r io.Reader) (Pipe, func() error) {
	reader, rwait := NewReaderPipe(r)
	buffer, bwait := NewBufferPipe(nil)

	pipe := streamPipe{
		reader: reader,
		buffer: buffer,
	}

	wait := func() error {
		err := rwait()
		if err != nil {
			buffer.Close()
			return err
		}

		return bwait()
	}
	return pipe, wait
}

func (pipe streamPipe) Pull() (string, func()) {
	select {
	case line, ok := <-pipe.reader.taskc:
		return pipe.reader.pull(line, ok)
	default:
	}

	select {
	case line, ok := <-pipe.reader.taskc:
		return pipe.reader.pull(line, ok)
	case line, ok := <-pipe.buffer.taskc:
		return pipe.buffer.pull(line, ok)
	}
}

func (pipe streamPipe) Push(queries []string) {
	pipe.buffer.Push(queries)
}

func (pipe streamPipe) Close() {
	pipe.reader.Close()
	pipe.buffer.Close()
}
