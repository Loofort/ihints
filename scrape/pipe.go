package scrape

import (
	"sync"
)

type MemPipe struct {
	taskc   chan string
	resultc chan []string
	closec  chan struct{}
	wg      *sync.WaitGroup
}

// creates and return memory pipe
// also return wait func that returns when pipe is done
func NewMemPipe(queries []string) (MemPipe, func()) {
	pipe := MemPipe{
		taskc:   make(chan string),
		resultc: make(chan []string),
		closec:  make(chan struct{}),
		wg:      new(sync.WaitGroup),
	}

	go pipe.taskLoop(queries)

	pipe.wg.Add(1)
	done := func() {
		pipe.wg.Done()
		pipe.wg.Wait()
		close(pipe.taskc)
	}

	return pipe, done
}

func (pipe MemPipe) taskLoop(queries []string) {
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

// Returns query and done func that should be called after Push new tasks if any.
// done indicates that the query processing is finished.
// done == nil if no more queries to process.
func (pipe MemPipe) Pull() (string, func()) {
	query, ok := <-pipe.taskc
	if !ok {
		return "", nil
	}

	pipe.wg.Add(1)
	done := func() { pipe.wg.Done() }
	return query, done
}

// Add new queries to the pipe
func (pipe MemPipe) Push(queries []string) {
	pipe.resultc <- queries
}

// closes the pipe
// it waits for the all in-flight tasks to be finished
func (pipe MemPipe) Close() {
	close(pipe.closec)
}
