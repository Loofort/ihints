package diff

import (
	"io"
	"strconv"

	"github.com/Loofort/xscrape/search"
)

type Difference struct {
	Num    int
	ID     string
	SubID  string
	Single bool
}

func (df Difference) String() string {
	new := "alive"
	if df.Num > 0 && df.Single {
		new = "new"
	}
	if df.Num < 0 && df.Single {
		new = "die"
	}
	num := strconv.Itoa(df.Num)
	return num + "\t" + new + "\t" + df.ID + "\t" + df.SubID
}

func Diff(r1, r2 io.Reader) ([]Difference, error) {
	ss1, err := search.FromReader(r1)
	if err != nil {
		return nil, err
	}

	ss2, err := search.FromReader(r2)
	if err != nil {
		return nil, err
	}

	search.Sort(ss1)
	search.Sort(ss2)

	var cmp int8 = 0
	var one, two search.Search
	var end1, end2 bool
	diffs := make([]Difference, 0, len(ss2))
	for len(ss1) > 0 || len(ss2) > 0 {
		if cmp <= 0 && !end1 {
			if len(ss1) == 0 {
				end1 = true
				cmp = 1
			} else {
				one, ss1 = ss1[0], ss1[1:]
			}
		}
		if cmp >= 0 && !end2 {
			if len(ss2) == 0 {
				end2 = true
				cmp = -1
			} else {
				two, ss2 = ss2[0], ss2[1:]
			}
		}

		if !end1 && !end2 {
			cmp = search.Compare(one, two)
		}

		var df Difference
		switch cmp {
		case -1: // one is higher
			df = Difference{
				Num:    -int(one.Position),
				ID:     one.BundleID,
				SubID:  one.Term,
				Single: true,
			}
		case 1: // two is higher
			df = Difference{
				Num:    int(two.Position),
				ID:     two.BundleID,
				SubID:  two.Term,
				Single: true,
			}
		default: // equal position
			df = Difference{
				Num:   int(two.Position) - int(one.Position),
				ID:    one.BundleID,
				SubID: one.Term,
			}
		}
		diffs = append(diffs, df)
	}

	return diffs, nil
}
