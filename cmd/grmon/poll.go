package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bcicen/grmon"
	"plist.jd.com/lc/lclogger"
)

type sortFn func(*grmon.Routine, *grmon.Routine) bool

var (
	client = &http.Client{Timeout: 10 * time.Second}

	sortKey = "num"
	sorters = map[string]sortFn{
		"num": func(r1, r2 *grmon.Routine) bool {
			return r1.Num < r2.Num
		},
		"state": func(r1, r2 *grmon.Routine) bool {
			return r1.State < r2.State
		},
	}
)

type Routines []*grmon.Routine

func (r Routines) Sort()              { sort.Sort(r) }
func (r Routines) Len() int           { return len(r) }
func (r Routines) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r Routines) Less(i, j int) bool { return sorters[sortKey](r[i], r[j]) }

func poll() (routines Routines, err error) {
	url := fmt.Sprintf("http://%s%s", *hostFlag, *endpointFlag)
	r, err := client.Get(url)
	if err != nil {
		return
	}
	defer r.Body.Close()

	// err = json.NewDecoder(r.Body).Decode(&routines)
	// lclogger.Warn(err, "AAAAAAA")
	// if err != nil {
	// 	return
	// }
	var p *grmon.Routine
	var buf bytes.Buffer

	body, err := ioutil.ReadAll(r.Body)
	lclogger.Warn(err, "AAAAAAA")
	if err != nil {
		return
	}
	_, err = buf.Write(body)
	lclogger.Warn(err, "AAAAAAA")
	if err != nil {
		return
	}
	for {
		line, err := buf.ReadString(newline)
		if err != nil {
			break
		}

		mg := statusRe.FindStringSubmatch(line)
		if len(mg) > 2 {
			// new routine block
			p = &grmon.Routine{}

			i, err := strconv.Atoi(mg[1])
			if err != nil {
				panic(err)
			}
			p.Num = i

			p.State = mg[2]
			routines = append(routines, p)
			continue
		}

		mg = createdRe.FindStringSubmatch(line)
		if len(mg) > 1 {
			p.CreatedBy = mg[1]
		}

		line = strings.Trim(line, "\n")
		if line != "" {
			p.Trace = append(p.Trace, line)
		}
	}

	sort.Sort(routines)
	return
}

var (
	newline   = byte(10)
	statusRe  = regexp.MustCompile("^goroutine\\s(\\d+)\\s\\[(.*)\\]:")
	createdRe = regexp.MustCompile("^created by (.*)")
)

type Routine struct {
	Num       int      `json:"no"`
	State     string   `json:"state"`
	CreatedBy string   `json:"created_by"`
	Trace     []string `json:"trace"`
}
