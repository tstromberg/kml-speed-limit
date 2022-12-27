package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var minAdjustedSpeed = 20.0
var speedRe = regexp.MustCompile(`Speed: ([\d\.]+) mph`)
var tableRe = regexp.MustCompile(`<tr><td><b>(.*?)</b>(.*?)</td></tr>`)
var destRe = regexp.MustCompile(`^        <name>(.*)</name>`)

type Result struct {
	Path                string
	Destination         string
	AverageSpeed        float64
	TravelSpeed         float64
	AdjustedTravelSpeed float64
	MaxSpeed            float64
	ModeSpeed           float64
	Table               map[string]string
}

func analyzeFile(f io.ReadCloser) Result {
	speeds := []float64{}
	speedMode := map[int]int{}
	table := map[string]string{}
	dest := ""

	var sum float64
	maxSpeed := 0.00
	scanner := bufio.NewScanner(f)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		matches := destRe.FindStringSubmatch(scanner.Text())
		if len(matches) > 0 && dest == "" {
			dest = matches[1]
			continue
		}
		matches = tableRe.FindStringSubmatch(scanner.Text())
		if len(matches) > 0 {
			k := matches[1]
			v := matches[2]
			table[k] = strings.TrimSpace(v)
			continue
		}

		matches = speedRe.FindStringSubmatch(scanner.Text())
		if len(matches) == 0 {
			continue
		}

		f, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			log.Printf("ignoring %q: %v", matches[1], err)
			continue
		}
		speeds = append(speeds, f)
		sum += f
		if f > maxSpeed {
			maxSpeed = f
		}
		speedMode[int(f)]++
	}

	modeSpeed := 0
	modeSpeedOcc := 0
	for k, v := range speedMode {
		if v > modeSpeedOcc {
			modeSpeed = k
			modeSpeedOcc = v
		}
	}

	travelBuffer := int(float64(len(speeds))*0.1) + 1
	midSection := speeds[travelBuffer : len(speeds)-travelBuffer]
	midSectionTotal := 0.00
	midSectionAdj := []float64{}
	midSectionAdjTotal := 0.00
	for _, s := range midSection {
		midSectionTotal += s
		if s < minAdjustedSpeed {
			continue
		}
		midSectionAdjTotal += s
		midSectionAdj = append(midSectionAdj, s)
	}

	return Result{
		AverageSpeed:        sum / float64(len(speeds)),
		TravelSpeed:         midSectionTotal / float64(len(midSection)),
		AdjustedTravelSpeed: midSectionAdjTotal / float64(len(midSectionAdj)),
		MaxSpeed:            maxSpeed,
		ModeSpeed:           float64(modeSpeed),
		Table:               table,
		Destination:         dest,
	}
}

func main() {
	rs := []Result{}

	for _, path := range os.Args[1:] {
		f, err := os.Open(path)
		if err != nil {
			fmt.Printf("read file: %v", err)
			os.Exit(1)
		}
		defer f.Close()
		r := analyzeFile(f)
		r.Path = path
		rs = append(rs, r)
	}

	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Table["Start Time"] < rs[j].Table["Start Time"]
	})

	for _, r := range rs {
		fmt.Printf("Start Time:            %s\n", r.Table["Start Time"])
		fmt.Printf("Path:                  %s\n", filepath.Base(r.Path))
		fmt.Printf("Destination:           %s\n", r.Destination)
		fmt.Printf("Distance:              %s\n", r.Table["Distance"])
		fmt.Printf("Average Speed:         %.2f mph\n", r.AverageSpeed)
		fmt.Printf("Travel Speed:          %.2f mph\n", r.TravelSpeed)
		fmt.Printf("Adjusted Travel Speed: %.2f mph\n", r.AdjustedTravelSpeed)
		fmt.Println()
	}
}
