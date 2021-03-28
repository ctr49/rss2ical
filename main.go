package main

import (
	"log"
	"net/http"
	"os"
	"bytes"
	"time"
	"regexp"
	"net"
	"net/url"
        _ "fmt"
	"github.com/emersion/go-ical"
	"github.com/mmcdole/gofeed"
)

type myRegexp struct {
  *regexp.Regexp
}

func (r *myRegexp) FindStringSubmatchMap(s string) map[string]string {
  captures := make(map[string]string)

  match := r.FindStringSubmatch(s)
  if match == nil {
      return captures
  }

  for i, name := range r.SubexpNames() {
      if i == 0 || name == "" {
          continue
      }
      
      captures[name] = match[i]

  }
  return captures
}

func rss2ical(w http.ResponseWriter, r *http.Request) {
  qp := r.URL.RawQuery
  if qp == "" {
    http.Error(w, "No parameters provided!", 400)
    return
  }
  var itemRegexp = myRegexp{regexp.MustCompile(`^Gremium:\s(?P<Gremium>.+)\sDatum:\s(?P<Datum>.+)\sZeit:\s(?P<Startzeit>.+)\sOrt:\s(?P<Ort>.+)$`)}
  var stripNonWord = regexp.MustCompile(`[^\w]`)
  feedUrl := qp
  validatedFeedUrl, validateErr := url.Parse(feedUrl)
  if validateErr != nil {
    http.Error(w, validateErr.Error(), 400)
    return
  }
  host := validatedFeedUrl.Host
  strippedHost, _, splitErr := net.SplitHostPort(host)
  if splitErr == nil {
    host = strippedHost
  }
  nowRuntime := time.Now()
  loc, locErr := time.LoadLocation("Europe/Berlin")
  if locErr != nil {
    panic(locErr)
  }
  fp := gofeed.NewParser()
  feed, fpErr := fp.ParseURL(feedUrl)
  if fpErr != nil {
    http.Error(w, fpErr.Error(), 400)
    return
  }
  cal := ical.NewCalendar()
  cal.Props.SetText(ical.PropVersion, "2.0")
  cal.Props.SetText(ical.PropProductID, "-//" + feed.Copyright + "//" + feed.Description + "//" + feed.Language)

  for _, currItem := range feed.Items {
    item := itemRegexp.FindStringSubmatchMap(currItem.Description)
    myDateString := item["Datum"] + " " + item["Startzeit"]
    myDate, DateErr := time.ParseInLocation("02.01.2006 15:04", myDateString, loc)
    if DateErr != nil {
      panic(DateErr)
    }
    event := ical.NewEvent()
    localUID := stripNonWord.ReplaceAllString(item["Gremium"] + item["Datum"], "")
    event.Props.SetText(ical.PropUID, localUID + "@" + host)
    event.Props.SetDateTime(ical.PropDateTimeStamp, nowRuntime)
    event.Props.SetText(ical.PropSummary, item["Gremium"])
    event.Props.SetText(ical.PropLocation, item["Ort"])
    for _, category := range currItem.Categories {
      event.Props.SetText(ical.PropCategories, category)
    }
    event.Props.SetDateTime(ical.PropDateTimeStart, myDate)
    cal.Children = append(cal.Children, event.Component)
  }
  var buf bytes.Buffer
    if bufErr := ical.NewEncoder(&buf).Encode(cal); bufErr != nil {
      log.Fatal(bufErr)
    }

  buf.WriteTo(w)

}

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		//log.Fatal("$PORT must be set")
                port = "3000"
	}

	http.HandleFunc("/Sitzungstermine.ics", rss2ical)

	if err := http.ListenAndServe(":" + port, nil); err != nil {
		panic(err)
	}
}
