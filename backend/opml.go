package main

import (
	"encoding/xml"
)

type OpmlDocument struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    OpmlHead `xml:"head"`
	Body    OpmlBody `xml:"body"`
}

type OpmlHead struct {
	Title string `xml:"title"`
}

type OpmlBody struct {
	Outlines []OpmlOutline `xml:"outline"`
}

type OpmlOutline struct {
	Text  string `xml:"text,attr"`
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
	URL   string `xml:"xmlUrl,attr"`
}
