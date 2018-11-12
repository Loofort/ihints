package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/alecthomas/kingpin.v2"
)

var priority = kingpin.Flag("priority", "set minimum desired hint priority").Default("1000").Short('p').Int16()
var cmd = kingpin.Arg("cmd", "one of the command [scrape|snap|diff]").Required().String()

func main() {
	kingpin.Parse()

	dir := "data/" + time.Now().Format(time.RFC3339) + "/"
	switch os.Args[1] {
	case "scrape":
		fmt.Printf("scrape hint for priority %d to folder %s\n", *priority, dir)
		scrape(int16(*priority), dir)
	}
}
