package scrape

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Loofort/xscrape/search"
)

type Pipe interface {
	Pull() (string, func())
}

// return true when no more query to scrape
func Iterate(client *http.Client, pipe Pipe, storage io.Writer, country string) (bool, error) {
	// get new query to proccess
	term, done := pipe.Pull()
	if done == nil {
		return true, nil
	}
	defer done()

	// scrape search from itunes
	apps, err := search.Scrape(client, term, country, 200)
	if err != nil {
		return false, fmt.Errorf("can't scrape '%s': %v", term, err)
	}

	// save search and apps
	if len(apps) == 0 {
		return false, nil
	}

	// save search
	b := search.ToBytes(term, apps)
	storage.Write(b)

	// todo:
	// save apps
	return false, nil
}
