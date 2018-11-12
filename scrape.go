package main

import (
	"log"
	"net/http"
	"os"
)

const (
	queryName = "query.txt"
	hintsName = "hints.tsv"
)

func scrape(minPriority int16, dir string) {
	err := initQueries(dir + queryName)
	mustNE(err)

	stg, wait := NewStorageNE(minPriority, dir+queryName, dir+hintsName)
	for i := 0; i < 10; i++ {
		go worker(stg)
	}
	wait()
}

func initQueries(filename string) error {
	ff, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	_, err = ff.Write(queries2bytes(generateQueries("")))
	if err != nil {
		return err
	}

	return ff.Close()
}

func worker(stg *Storage) {
	for {
		// get new query to proccess
		q, idx, done := stg.Get()
		if done == nil {
			return
		}

		// scrape hints from itunes
		hints, err := GetHints(q, http.DefaultClient)
		if err != nil {
			log.Printf("can't scrape '%s': %v", q, err)
		}

		// save results, it produces other queries if needed
		err = stg.Set(q, idx, hints)
		if err != nil {
			log.Printf("unexpected hints result for '%s': %v", q, err)
		}

		done()
	}
}
