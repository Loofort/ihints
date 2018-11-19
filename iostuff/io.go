package iostuff

import (
	"bufio"
	"io"
	"os"
	"sync"
)

// return reader from filename if provided, otherwise from stdin
func InputReader(filename string) (io.ReadCloser, error) {
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

func inputlines(filename string) ([]string, error) {
	r, err := InputReader(filename)
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

func OutputWriter(filename string) (io.WriteCloser, error) {
	if filename != "" {
		fh, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		return &SafeWriter{WriteCloser: fh}, nil
	}

	return &SafeWriter{WriteCloser: os.Stdout}, nil
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
