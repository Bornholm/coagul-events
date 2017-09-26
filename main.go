package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"html/template"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const separator = ";"
const dateFormat = "2006-01-02"

var startDateStr string
var endDateStr string
var eventsURLTpl *template.Template

// ([0-9]{2}/[0-9]{2}/[0-9]{4})?[ \n]+$
var eventRE = regexp.MustCompile("(?s)^[ \n]+([^\n]+)(.*)$")
var timingRE = regexp.MustCompile(`([0-9]{2}/[0-9]{2}/[0-9]{4})(( - [0-9]{2}:[0-9]{2})|( \(Toute la journée\)))( - ([0-9]{2}:[0-9]{2}))?`)

func init() {
	flag.StringVar(&startDateStr, "start", time.Now().Format(dateFormat), "The start date (inclusive)")
	flag.StringVar(&endDateStr, "end", time.Now().Format(dateFormat), "The end date (exclusive)")
}

func main() {

	var err error

	flag.Parse()

	eventsURLTpl, err = template.New("eventsURL").Parse("http://coagul.org/drupal/calendrier/{{.Year}}-{{.Month}}-{{.Day}}")
	if err != nil {
		panic(err)
	}

	startDate, err := time.Parse(dateFormat, startDateStr)
	if err != nil {
		panic(err)
	}

	endDate, err := time.Parse(dateFormat, endDateStr)
	if err != nil {
		panic(err)
	}

	csvWriter := csv.NewWriter(os.Stdout)
	csvWriter.Comma = []rune(separator)[0]

	csvWriter.Write([]string{"TYPE", "TITRE", "DATE_DEBUT", "HEURE_DEBUT", "DATE_FIN", "HEURE_FIN", "URL"})

	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {

		url := getEventsURLForDate(d)

		doc, err := goquery.NewDocument(url)
		if err != nil {
			panic(err)
		}

		// Recherche toutes les entrées d'évènements dans la page de la journée
		doc.Find(".view-item-calendar").Each(func(i int, sel *goquery.Selection) {

			text := sel.Text()
			matches := eventRE.FindStringSubmatch(text)

			title := strings.Trim(matches[1], " \n")
			timing := strings.Trim(matches[2], " \n")

			timeMatches := timingRE.FindAllStringSubmatch(timing, 2)

			var startDate string
			var startTime string
			var endDate string
			var endTime string

			if len(timeMatches) > 0 {
				startDate = timeMatches[0][1]
				startTime = strings.Trim(strings.Replace(timeMatches[0][2], " - ", "", 1), " ()")
			}

			if len(timeMatches) > 1 {
				endDate = timeMatches[1][1]
				endTime = strings.Trim(strings.Replace(timeMatches[1][2], " - ", "", 1), " ()")
			}

			eventType := "autre"
			lowerTitle := strings.ToLower(title)

			switch {
			case strings.Contains(lowerTitle, "cartographie"):
				eventType = "carto"
			case strings.Contains(lowerTitle, "hacklab") || strings.Contains(lowerTitle, "hackerspace") || strings.Contains(lowerTitle, "fablab"):
				eventType = "hacklab"
			case strings.Contains(lowerTitle, "permanence"):
				eventType = "permanence"
			}

			if err := csvWriter.Write([]string{eventType, title, startDate, startTime, endDate, endTime, url}); err != nil {
				panic(err)
			}

		})

		csvWriter.Flush()

	}

}

func getEventsURLForDate(date time.Time) string {
	var urlBuffer bytes.Buffer
	eventsURLTpl.Execute(&urlBuffer, struct {
		Day   string
		Year  string
		Month string
	}{Day: date.Format("02"), Month: date.Format("01"), Year: date.Format("2006")})
	return urlBuffer.String()
}
