package search

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"time"
)

type Search struct {
	Position byte
	BundleID string
	Term     string
}

func (search Search) String() string {
	pos := strconv.Itoa(int(search.Position))
	return pos + "\t" + search.BundleID + "\t" + search.Term
}

type serp struct {
	ResultCount int
	Results     []App
}

func Scrape(client *http.Client, term, country string) ([]App, error) {
	// https://itunes.apple.com/search?country=us&entity=software&term=flappy
	// skip media and limit (=50) and attribute
	v := url.Values{}
	v.Set("entity", "software")
	v.Set("term", term)
	if country != "" {
		v.Set("country", country)
	}
	url := "https://itunes.apple.com/search?" + v.Encode()

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// expect status 200
	if resp.StatusCode != http.StatusOK {
		body, err := httputil.DumpResponse(resp, true)
		if err != nil {
			errmsg := fmt.Sprintf("cant dump resp: %v", err)
			body = []byte(errmsg)
		}
		return nil, fmt.Errorf("unexpected http status %d; dump: %s", resp.StatusCode, body)
	}

	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()

	se := serp{}
	if err := dec.Decode(&se); err != nil {
		//body, err := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("unable parse resp: %v; url: %s", err, url)
	}

	return se.Results, nil
}

type App struct {
	BundleID                           string    `json:"bundleId"`
	ScreenshotUrls                     []string  `json:"screenshotUrls"`
	IpadScreenshotUrls                 []string  `json:"ipadScreenshotUrls"`
	AppletvScreenshotUrls              []string  `json:"appletvScreenshotUrls"`
	ArtworkURL60                       string    `json:"artworkUrl60"`
	ArtworkURL512                      string    `json:"artworkUrl512"`
	ArtworkURL100                      string    `json:"artworkUrl100"`
	ArtistViewURL                      string    `json:"artistViewUrl"`
	SupportedDevices                   []string  `json:"supportedDevices"`
	IsGameCenterEnabled                bool      `json:"isGameCenterEnabled"`
	Kind                               string    `json:"kind"`
	Features                           []string  `json:"features"`
	Advisories                         []string  `json:"advisories"`
	AverageUserRatingForCurrentVersion float64   `json:"averageUserRatingForCurrentVersion"`
	TrackCensoredName                  string    `json:"trackCensoredName"`
	LanguageCodesISO2A                 []string  `json:"languageCodesISO2A"`
	FileSizeBytes                      string    `json:"fileSizeBytes"`
	SellerURL                          string    `json:"sellerUrl"`
	ContentAdvisoryRating              string    `json:"contentAdvisoryRating"`
	UserRatingCountForCurrentVersion   int       `json:"userRatingCountForCurrentVersion"`
	TrackViewURL                       string    `json:"trackViewUrl"`
	TrackContentRating                 string    `json:"trackContentRating"`
	CurrentVersionReleaseDate          time.Time `json:"currentVersionReleaseDate"`
	SellerName                         string    `json:"sellerName"`
	ReleaseDate                        time.Time `json:"releaseDate"`
	PrimaryGenreName                   string    `json:"primaryGenreName"`
	TrackID                            int       `json:"trackId"`
	TrackName                          string    `json:"trackName"`
	ReleaseNotes                       string    `json:"releaseNotes"`
	GenreIds                           []string  `json:"genreIds"`
	FormattedPrice                     string    `json:"formattedPrice"`
	PrimaryGenreID                     int       `json:"primaryGenreId"`
	IsVppDeviceBasedLicensingEnabled   bool      `json:"isVppDeviceBasedLicensingEnabled"`
	MinimumOsVersion                   string    `json:"minimumOsVersion"`
	Currency                           string    `json:"currency"`
	WrapperType                        string    `json:"wrapperType"`
	Version                            string    `json:"version"`
	Description                        string    `json:"description"`
	ArtistID                           int       `json:"artistId"`
	ArtistName                         string    `json:"artistName"`
	Genres                             []string  `json:"genres"`
	Price                              float64   `json:"price"`
	AverageUserRating                  float64   `json:"averageUserRating"`
	UserRatingCount                    int       `json:"userRatingCount"`
}

func FromApps(term string, apps []App) []Search {
	ss := make([]Search, 0, len(apps))
	for i, app := range apps {

		search := Search{
			Position: byte(i),
			BundleID: app.BundleID,
			Term:     term,
		}
		ss = append(ss, search)
	}

	return ss
}

func ToBytes(term string, apps []App) []byte {
	b := new(bytes.Buffer)

	fmt.Fprintf(b, "%s\t", term)
	for i, app := range apps {
		fmt.Fprintf(b, "%s", app.BundleID) // can't be error
		if i < len(apps)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")
	return b.Bytes()
}

func Sort(ss []Search) {
	sort.Slice(ss, func(i, j int) bool {
		if ss[i].Term == ss[j].Term {
			return ss[i].BundleID < ss[j].BundleID
		}
		return ss[i].Term < ss[j].Term
	})
}

func FromReader(reader io.Reader) ([]Search, error) {
	r := bufio.NewReader(reader)

	ss := []Search{}
	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		pices := bytes.SplitN(line, []byte{'\t'}, 3)
		if len(pices) != 3 {
			return nil, fmt.Errorf("incorrect line: %s", line)
		}

		pos, err := strconv.Atoi(string(pices[0]))
		if err != nil {
			return nil, fmt.Errorf("bad position: %v", err)
		}

		search := Search{
			Position: byte(pos),
			BundleID: string(pices[1]),
			Term:     string(pices[2][:len(pices[2])-1]), // cutoff last \n symbol
		}
		ss = append(ss, search)
	}

	return ss, nil
}
