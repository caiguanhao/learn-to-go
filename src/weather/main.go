package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type Detail struct {
	Text string `json:"description"`
}

type Coord struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Temperature struct {
	Max float64 `json:"max"`
	Min float64 `json:"min"`
}

type List struct {
	Time        int64       `json:"dt"`
	Pressure    float64     `json:"pressure"`
	Humidity    float64     `json:"humidity"`
	Speed       float64     `json:"speed"`
	Degree      float64     `json:"deg"`
	Clouds      float64     `json:"clouds"`
	Rain        float64     `json:"rain"`
	Details     []Detail    `json:"weather"`
	Temperature Temperature `json:"temp"`
}

type City struct {
	Name  string `json:"name"`
	Coord Coord  `json:"coord"`
}

type Weather struct {
	City  City   `json:"city"`
	Lists []List `json:"list"`
}

func formatDate(unixTimeStamp int64) string {
	ti := time.Unix(unixTimeStamp, 0)
	return fmt.Sprintf("%d-%02d-%02d (%s)",
		ti.Year(), ti.Month(), ti.Day(), ti.Weekday().String()[0:3])
}

// ssh/terminal/util.go GetSize()
func getTerminalSize() (width, height int, err error) {
	var dimensions [4]uint16
	if _, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&dimensions)),
		0, 0, 0); err != 0 {
		return -1, -1, err
	}
	return int(dimensions[1]), int(dimensions[0]), nil
}

func printHorizontalLine(width int) {
	fmt.Println(strings.Replace(fmt.Sprintf("%*s", width, " "), " ", "-", -1))
}

func toCelsius(degree float64) float64 {
	return degree - 273.15
}

func main() {
	city := flag.String("city", "Daliang", "<name>     Name of the city")
	fake := flag.Bool("fake", false, "           Use fake data")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTION] [[of] city]\n\n", os.Args[0])
		flag.VisitAll(func(flag *flag.Flag) {
			switch flag.DefValue {
			case "true", "false":
				fmt.Fprintf(os.Stderr, "  --%s %s\n", flag.Name, flag.Usage)
			default:
				fmt.Fprintf(os.Stderr, "  --%s %s, default is %s\n",
					flag.Name, flag.Usage, flag.DefValue)
			}
		})
	}
	flag.Parse()

	termWidth, _, _ := getTerminalSize()

	pgfmt := [9]string{
		"  Time      %s",
		"  Status    %s",
		"  Clouds    %s",
		"  Degree    %s",
		"  Humidity  %s",
		"  Pressure  %s",
		"  Rain      %s",
		"  Speed     %s",
		"  Temp      %s",
	}

	api := "http://api.openweathermap.org/data/2.5/forecast/daily?q=%s"

	var res *http.Response
	var body []byte
	var err error

	if !*fake {
		res, err = http.Get(fmt.Sprintf(api, *city))
	}
	if !*fake && err == nil {
		body, err = ioutil.ReadAll(res.Body)
		defer res.Body.Close()
	} else {
		body, err = ioutil.ReadFile("data/fake-weather.json")
	}

	if err != nil {
		panic(err)
	}

	weather := &Weather{}
	json.Unmarshal(body, &weather)

	if weather.City.Name == "" {
		fmt.Fprintf(os.Stderr, "No weather forecast for city %s!\n", *city)
		os.Exit(1)
	}

	title := fmt.Sprintf("Weather Forecast for %s (%0.5f, %0.5f)",
		weather.City.Name,
		weather.City.Coord.Lat,
		weather.City.Coord.Lon)

	offset := int(math.Max(math.Floor(float64(termWidth-len(title))/2), 0))

	printHorizontalLine(termWidth)
	fmt.Printf("%*s%s\n", offset, "", title)

	listLen := len(weather.Lists)

	colWidth := 34.0
	cols := int(math.Max(math.Floor(float64(termWidth)/colWidth), 1.0))
	rows := int(math.Ceil(float64(listLen) / float64(cols)))

	data := make([][9]string, listLen)

	for i := range weather.Lists {
		list := weather.Lists[i]
		data[i] = [9]string{
			formatDate(list.Time),
			list.Details[0].Text,
			fmt.Sprintf("%.0f", list.Clouds),
			fmt.Sprintf("%.0f", list.Degree),
			fmt.Sprintf("%.0f", list.Humidity),
			fmt.Sprintf("%.2f", list.Pressure),
			fmt.Sprintf("%.0f", list.Rain),
			fmt.Sprintf("%.2f", list.Speed),
			fmt.Sprintf("%.2f Hi / %.2f Lo",
				toCelsius(list.Temperature.Max),
				toCelsius(list.Temperature.Min)),
		}
	}

	for i := 0; i < rows; i++ {
		printHorizontalLine(termWidth)
		for j := 0; j < len(pgfmt); j++ {
			for k := 0; k < cols; k++ {
				index := i*cols + k
				if index >= listLen {
					continue
				}
				o := fmt.Sprintf(pgfmt[j], data[index][j])
				f := int(colWidth) - len(o)
				if f > 0 {
					fmt.Printf("%s%*s", o, f, " ")
				} else {
					fmt.Print(o)
				}
			}
			fmt.Print("\n")
		}
	}
}
