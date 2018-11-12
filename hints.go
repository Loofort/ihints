package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
)

const ihost = "https://search.itunes.apple.com/"

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
