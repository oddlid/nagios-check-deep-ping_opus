package main

import (
	"encoding/xml"
)

type PingResponse struct {
	XMLName     xml.Name      `xml:"PingResponse"`
	Application []Application `xml:"application"`
	Err         error
}

type Application struct {
	XMLName       xml.Name       `xml:"application"`
	LongName      string         `xml:"long-name,attr"`
	ShortName     string         `xml:"short-name,attr"`
	Version       string         `xml:"version,attr"`
	FailureReason string         `xml:"failureReason"`
	EndPoint      string         `xml:"endpoint"`
	Success       bool           `xml:"success"`
	Skipped       bool           `xml:"skipped"`
	ResponseTime  float64        `xml:"responsetime"`
	Dependencies  []Dependencies `xml:"dependencies"`
}

type Dependencies struct {
	XMLName        xml.Name         `xml:"dependencies"`
	Infrastructure []Infrastructure `xml:"infrastructure"`
	Application    []Application    `xml:"application"`
}

type Infrastructure struct {
	XMLName      xml.Name `xml:"infrastructure"`
	Name         string   `xml:"name,attr"`
	Type         string   `xml:"type,attr"`
	Success      bool     `xml:"success"`
	ResponseTime float64  `xml:"responsetime"`
}

