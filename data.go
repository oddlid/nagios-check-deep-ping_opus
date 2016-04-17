package main

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type PingResponse struct {
	XMLName      xml.Name      `xml:"PingResponse" json:",omitempty"`
	Application  []Application `xml:"application,omitempty" json:",omitempty"`
	Err          error         `json:"error,omitempty"`
	ResponseTime float64       `json:",omitempty"`
	PP_Prefix    string        `json:",omitempty"`
}

type Application struct {
	XMLName       xml.Name       `xml:"application" json:",omitempty"`
	LongName      string         `xml:"long-name,attr" json:",omitempty"`
	ShortName     string         `xml:"short-name,attr" json:",omitempty"`
	Version       string         `xml:"version,attr" json:",omitempty"`
	FailureReason string         `xml:"failureReason,omitempty" json:",omitempty"`
	EndPoint      string         `xml:"endpoint,omitempty" json:",omitempty"`
	Success       bool           `xml:"success" json:",omitempty"`
	Skipped       bool           `xml:"skipped,omitempty" json:",omitempty"`
	ResponseTime  float64        `xml:"responsetime" json:",omitempty"`
	Dependencies  []Dependencies `xml:"dependencies,omitempty" json:",omitempty"`
}

type Dependencies struct {
	XMLName        xml.Name         `xml:"dependencies" json:",omitempty"`
	Infrastructure []Infrastructure `xml:"infrastructure,omitempty" json:",omitempty"`
	Application    []Application    `xml:"application,omitempty" json:",omitempty"`
}

type Infrastructure struct {
	XMLName      xml.Name `xml:"infrastructure" json:",omitempty"`
	Name         string   `xml:"name,attr" json:",omitempty"`
	Type         string   `xml:"type,attr" json:",omitempty"`
	Success      bool     `xml:"success" json:",omitempty"`
	ResponseTime float64  `xml:"responsetime" json:",omitempty"`
}

func (p PingResponse) String() string {
	if p.PP_Prefix == "" {
		p.PP_Prefix = "\t"
	}
	return p.pp(p.PP_Prefix, 0)
}

func _pp(prefix, key, val string, align, level int) string {
	pf := strings.Repeat(prefix, level)
	f := fmt.Sprintf("%s%s%d%s", pf, "%-", align, "s : %s\n")
	return fmt.Sprintf(f, key, val)
}

func (i Infrastructure) pp(prefix string, level int) string {
	w := 12
	str := _pp(prefix, "Name", i.Name, w, level)
	str += _pp(prefix, "Type", i.Type, w, level)
	str += _pp(prefix, "Success", fmt.Sprintf("%t", i.Success), w, level)
	str += _pp(prefix, "Responsetime", fmt.Sprintf("%v", i.ResponseTime), w, level)
	return str
}

func (a Application) pp(prefix string, level int) string {
	w := 13
	str := _pp(prefix, "LongName", a.LongName, w, level)
	str += _pp(prefix, "ShortName", a.ShortName, w, level)
	str += _pp(prefix, "Version", a.Version, w, level)
	str += _pp(prefix, "FailureReason", a.FailureReason, w, level)
	str += _pp(prefix, "EndPoint", a.EndPoint, w, level)
	str += _pp(prefix, "Success", fmt.Sprintf("%t", a.Success), w, level)
	str += _pp(prefix, "Skipped", fmt.Sprintf("%t", a.Skipped), w, level)
	str += _pp(prefix, "ResponseTime", fmt.Sprintf("%f", a.ResponseTime), w, level)

	if len(a.Dependencies) > 0 {
		for i := range a.Dependencies {
			str += _pp(prefix, "Dependencies", "", w, level)
			str += a.Dependencies[i].pp(prefix, level+1)
		}
	}

	return str
}

func (d Dependencies) pp(prefix string, level int) string {
	w := 14
	var str string
	if len(d.Infrastructure) > 0 {
		for i := range d.Infrastructure {
			str += _pp(prefix, "Infrastructure", "", w, level)
			str += d.Infrastructure[i].pp(prefix, level+1)
		}
	}
	if len(d.Application) > 0 {
		for j := range d.Application {
			str += _pp(prefix, "Application", "", w, level)
			str += d.Application[j].pp(prefix, level+1)
		}
	}
	return str
}

func (p PingResponse) pp(prefix string, level int) string {
	w := 12
	str := "=== BEGIN: PingResponse ===\n"
	str += _pp(prefix, "ResponseTime", fmt.Sprintf("%f", p.ResponseTime), w, level)
	str += _pp(prefix, "Error", fmt.Sprintf("%v", p.Err), w, level)
	for i := range p.Application {
		str += _pp(prefix, "Application", "", w, level)
		str += p.Application[i].pp(prefix, level + 1)
	}
	str += "=== END: PingResponse ===\n"
	return str
}

func (d Dependencies) Success() bool {
	for i := range d.Infrastructure {
		if !d.Infrastructure[i].Success {
			return false
		}
	}
	for j := range d.Application {
		if !d.Application[j].Success {
			return false
		}
	}
	return true
}

func (p PingResponse) Success() bool {
	for i := range p.Application {
		if !p.Application[i].Success {
			return false
		}
		for j := range p.Application[i].Dependencies {
			if !p.Application[i].Dependencies[j].Success() {
				return false
			}
		}
	}
	return true
}
