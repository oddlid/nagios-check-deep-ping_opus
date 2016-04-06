package main

import (
	"encoding/xml"
)

type PingResponse struct {
	XMLName     xml.Name      `xml:"PingResponse"`
	Application []Application `xml:"PingResponse>application"`
}

type Application struct {
	XMLName       xml.Name       `xml:"application"`
	LongName      string         `xml:"long-name,attr"`
	ShortName     string         `xml:"short-name,attr"`
	Version       string         `xml:"version,attr"`
	FailureReason string         `xml:"application>failureReason"`
	EndPoint      string         `xml:"application>endpoint"`
	Success       bool           `xml:"application>success"`
	Skipped       bool           `xml:"application>skipped"`
	ResponseTime  float64        `xml:"application>responsetime"`
	Dependencies  []Dependencies `xml:"application>dependencies"`
}

type Dependencies struct {
	XMLName        xml.Name         `xml:"dependencies"`
	Infrastructure []Infrastructure `xml:"infrastructure"`
	Application    []Application    `xml:"dependencies>application"`
}

type Infrastructure struct {
	XMLName      xml.Name `xml:"infrastructure"`
	Name         string   `xml:"name,attr"`
	Type         string   `xml:"type,attr"`
	Success      bool     `xml:"infrastructure>success"`
	ResponseTime float64  `xml:"infrastructure>responsetime"`
}

