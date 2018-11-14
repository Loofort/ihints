package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const file = "data/2018-11-08/index.res"

const index = 127470

//const index = 991

func TestT(t *testing.T) {
	f, err := os.OpenFile(file, os.O_RDONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	offset := 0
	e := 0

	for i := 0; true; i++ {
		p := make([]byte, 2)
		_, err := r.Read(p)
		require.NoError(t, err)
		m := binary.LittleEndian.Uint16(p)
		mark := int16(m)

		switch {
		case mark < 0 && mark > -50:
			offset += int(-mark)
		case mark > 0:
			offset += 50
		case mark == 0:
			e++
		case mark == zeroPriority:
			offset += 50
		case mark == noHints:
		default:
			fmt.Printf("forgot about %d\n", mark)
		}

		if offset >= index {
			fmt.Printf("line %d (issues %d)\n", i, e)
			return
		}
	}
}
