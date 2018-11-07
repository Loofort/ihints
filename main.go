package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

const ihost = "https://search.itunes.apple.com/"

func main() {
	minPriority := int16(1000)

	hintsc := make(chan []Hint)
	go WriteHints(hintsc)

	resultc := make(chan Result)
	go writeResult(resultc)

	taskc := make(chan Task)
	queriesc := make(chan []string)

	// produce tasks
	f, err := os.OpenFile("query.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	wtireQ(f, generateQueries(""))
	f.Close()

	go writeQueries(f, queriesc)

	for i := 0; i < 1; i++ {
		go worker(minPriority, taskc, hintsc, resultc, queriesc)
	}

	r := bufio.NewReader(f)
	i := uint64(0)
	for {
		line, isPrefix, err := r.ReadLine()
		if err != nil {
			log.Printf("task error: %v\n", err)
			return
		}

		if isPrefix {
			errmsg := fmt.Sprintf("line prefix, index=%d query=%s", i, line)
			panic(errmsg)
		}

		fmt.Printf("task %d : %s\n", i, line)
		taskc <- Task{string(line), i}
		i++

	}

}

func writeQueries(f *os.File, queriesc chan []string) {
	for queries := range queriesc {
		wtireQ(f, queries)
	}
}

func wtireQ(f *os.File, queries []string) {
	for _, query := range queries {
		n, err := f.WriteString(query + "\n")
		if err != nil {
			panic(err)
		}
		fmt.Printf("writed %d:%s\n", n, query)
	}
	f.Sync()
}

func WriteHints(hintsc chan []Hint) {
	f, err := os.OpenFile("hints.tsv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for hints := range hintsc {
		for _, hint := range hints {
			str := fmt.Sprintf("%d\t%s\n", hint.Priority, hint.Term)
			if _, err := f.Write([]byte(str)); err != nil {
				panic(err)
			}
		}
	}
}

func writeResult(resultc chan Result) {
	f, err := os.OpenFile("index.res", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for result := range resultc {
		_ = result
		/*
			offset := result.Index * 2
			ret, err := f.Seek(offset, 0)
			if err != nil {
				panic(err)
			}

			b := make([]byte, 2)
			binary.LittleEndian.PutUint16(b, uint16(result.Mark))
			if _, err := f.Write(b); err != nil {
				panic(err)
			}
		*/
	}
}

/*
func Mem(taskc chan Task, hintsc chan []Hint, resultc chan Result, queriesc chan []string) {

	memHints = []Hint
	for hints := range hintsc {
		memHints = append(memHints, hints...)
	}

	for result := range resultc {
	}

}
*/

/********************************************************************************************************************************/

type Task struct {
	Query string
	Index uint64
}
type Result struct {
	Mark  int16
	Index uint64
}

func worker(minPriority int16, taskc chan Task, hintsc chan []Hint, resultc chan Result, queriesc chan []string) {

	for task := range taskc {
		fmt.Printf("q: %v\n", task.Query)
		hints, err := GetHints(task.Query, http.DefaultClient)
		if err != nil {
			// TODO
			log.Printf("GetHints error for q %s : %v", task.Query, err)
			continue
		}

		// save hints
		if len(hints) > 0 {
			hintsc <- hints
		}

		// save progress
		m, err := resultMark(hints)
		if err != nil {
			// TODO
			log.Printf("Algo error for q %s : %v", task.Query, err)
			continue
		}
		resultc <- Result{m, task.Index}

		// generate new queries
		if m > minPriority {
			queries := generateQueries(task.Query)
			queriesc <- queries
		}
	}
}

const letterBytes = " abcdefghijklmnopqrstuvwxyz"

func generateQueries(q string) []string {
	qs := make([]string, 0, len(letterBytes))
	for _, b := range letterBytes {
		gen := q + string(b)
		qs = append(qs, gen)
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
func resultMark(hints []Hint) (int16, error) {
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

var re = regexp.MustCompile(`\r?\n`)

func GetHints(q string, client *http.Client) ([]Hint, error) {
	v := url.Values{}
	v.Set("media", "software")
	v.Set("q", q)
	url := ihost + "/WebObjects/MZSearchHints.woa/wa/hints?" + v.Encode()

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	sh := XMLShit{}
	if err := xml.Unmarshal(body, &sh); err != nil {
		return nil, err
	}

	hints, err := sh.GetHints()
	if err != nil {
		body = re.ReplaceAll(body, []byte("\\n"))
		return nil, fmt.Errorf("%v: %s", err, body)
	}

	return hints, nil
}

type Hint struct {
	Term     string
	Priority int16
}

type XMLShit struct {
	Key    []string `xml:"dict>key"`
	String []string `xml:"dict>string"`
	Dict   []struct {
		Key     []string `xml:"key"`
		String  []string `xml:"string"`
		Integer []int16  `xml:"integer"`
	} `xml:"dict>array>dict"`
}

func (sh XMLShit) GetHints() ([]Hint, error) {
	// check valid format
	if len(sh.Key) != 2 || len(sh.String) != 1 ||
		sh.Key[0] != "title" || sh.Key[1] != "hints" || sh.String[0] != "Suggestions" {
		return nil, fmt.Errorf("invalid xml envelop")
	}

	hints := make([]Hint, 0, len(sh.Dict))
	for i, dict := range sh.Dict {
		// check valid format
		if len(dict.Key) != 3 || len(dict.String) != 2 || len(dict.Integer) != 1 ||
			dict.Key[0] != "term" || dict.Key[1] != "priority" || dict.Key[2] != "url" ||
			dict.String[1][0:len(ihost)] != ihost {
			return nil, fmt.Errorf("invalid xml dict %d", i)
		}

		hint := Hint{
			Term:     dict.String[0],
			Priority: dict.Integer[0],
		}
		hints = append(hints, hint)
	}

	return hints, nil
}
