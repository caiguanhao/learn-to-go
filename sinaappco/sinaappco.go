package main

/**
 * Since Google is completely blocked in China, if you don't want to use
 * VPN or other things and just want a fast and direct way to use Google
 * services, run this go script and it will return a string that can be
 * added as a search provider in Google Chrome Settings.
 */

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"strings"
)

func main() {
	doc, err := goquery.NewDocument("https://sinaapp.co/")
	if err != nil {
		panic(err)
	}
	noscript := bytes.NewReader([]byte(doc.Find("noscript").Text()))
	doc, _ = goquery.NewDocumentFromReader(noscript)
	link, _ := doc.Find("iframe").Attr("src")
	fmt.Printf("%s/search?newwindow=1&safe=off&hl=en-US&q=%%s\n",
		strings.TrimRight(link, "/"))
}
