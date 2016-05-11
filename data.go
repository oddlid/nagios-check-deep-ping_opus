package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

const (
	T_AP string = "Application"
	T_DE string = "Dependencies"
	T_DP string = "  " // Default Prefix, for pretty printing
	T_EP string = "EndPoint"
	T_FR string = "FailureReason"
	T_HC string = "HTTPCode"
	T_IS string = "Infrastructure"
	T_LN string = "LongName"
	T_NA string = "Name"
	T_PR string = "PingResponse"
	T_RT string = "ResponseTime"
	T_SK string = "Skipped"
	T_SN string = "ShortName"
	T_SU string = "Success"
	T_TY string = "Type"
	T_VS string = "Version"
)

// just internal stats for how many pretty print calls are actually made. For fun...
var pp_count int = 0

type Succeeder interface {
	Ok() bool
}

type PingResponse struct {
	XMLName      xml.Name      `xml:"PingResponse" json:",omitempty"`
	Application  []Application `xml:"application,omitempty" json:",omitempty"`
	Err          error         `json:"error,omitempty"`
	ResponseTime float64       `json:",omitempty"`
	PP_Prefix    string        `json:",omitempty"`
	HTTPCode     int           `json:",omitempty"`
	URL          string        `json:",omitempty"`
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
	var buf bytes.Buffer
	p.Dump(&buf)
	return buf.String()
}

func (p PingResponse) Dump(w io.Writer) {
	if p.PP_Prefix == "" {
		p.PP_Prefix = T_DP
	}
	p.pp(w, p.PP_Prefix, 0)
}

func (p PingResponse) DumpJSON(w io.Writer, indent bool) (int, error) {
	if p.PP_Prefix == "" {
		p.PP_Prefix = T_DP
	}
	var jbytes []byte
	var err error
	if !indent {
		jbytes, err = json.Marshal(p)
	} else {
		jbytes, err = json.MarshalIndent(p, "", p.PP_Prefix)
	}
	if err != nil {
		return 0, err
	}
	return w.Write(jbytes)
}

// _pp left-pads a string with <prefix> repeated <level> times,
// then right-pads the word <key> up to <align> length, then prints " : <val>\n"
func _pp(w io.Writer, prefix, key, val string, align, level int) {
	fmt.Fprintf(w, fmt.Sprintf("%s%s%d%s", strings.Repeat(prefix, level), "%-", align, "s : %s\n"), key, val)
	pp_count++
}

func (i Infrastructure) pp(w io.Writer, prefix string, level int) {
	p := func(k, v string) {
		_pp(w, prefix, k, v, 12, level)
	}
	p(T_NA, i.Name)
	p(T_TY, i.Type)
	p(T_SU, fmt.Sprintf("%t", i.Success))
	p(T_RT, fmt.Sprintf("%f", i.ResponseTime))
}

func (a Application) pp(w io.Writer, prefix string, level int) {
	p := func(k, v string) {
		_pp(w, prefix, k, v, 13, level)
	}
	p(T_LN, a.LongName)
	p(T_SN, a.ShortName)
	p(T_VS, a.Version)
	p(T_FR, a.FailureReason)
	p(T_EP, a.EndPoint)
	p(T_SU, fmt.Sprintf("%t", a.Success))
	p(T_SK, fmt.Sprintf("%t", a.Skipped))
	p(T_RT, fmt.Sprintf("%f", a.ResponseTime))

	if len(a.Dependencies) > 0 {
		for i := range a.Dependencies {
			p(T_DE, "")
			a.Dependencies[i].pp(w, prefix, level+1)
		}
	}
}

func (d Dependencies) pp(w io.Writer, prefix string, level int) {
	p := func(k, v string) {
		_pp(w, prefix, k, v, 14, level)
	}
	if len(d.Infrastructure) > 0 {
		for i := range d.Infrastructure {
			p(T_IS, "")
			d.Infrastructure[i].pp(w, prefix, level+1)
		}
	}
	if len(d.Application) > 0 {
		for j := range d.Application {
			p(T_AP, "")
			d.Application[j].pp(w, prefix, level+1)
		}
	}
}

func (pr PingResponse) pp(w io.Writer, prefix string, level int) {
	p := func(k, v string) {
		_pp(w, prefix, k, v, 12, level)
	}
	fmt.Fprintf(w, "=== BEGIN: %s =====\n", T_PR)
	p("URL", pr.URL)
	p(T_HC, fmt.Sprintf("%d", pr.HTTPCode))
	p(T_RT, fmt.Sprintf("%f", pr.ResponseTime))
	p("Error", fmt.Sprintf("%v", pr.Err))
	for i := range pr.Application {
		p(T_AP, "")
		pr.Application[i].pp(w, prefix, level+1)
	}
	fmt.Fprintf(w, "===== END: %s =====\n", T_PR)
}

// Implement the Succeeder interface

func (p PingResponse) Ok() bool {
	for i := range p.Application {
		if !p.Application[i].Ok() {
			return false
		}
	}
	return true
}

func (a Application) Ok() bool {
	for d := range a.Dependencies {
		if !a.Dependencies[d].Ok() {
			return false
		}
	}
	return a.Success
}

func (d Dependencies) Ok() bool {
	for i := range d.Infrastructure {
		if !d.Infrastructure[i].Ok() {
			return false
		}
	}
	for a := range d.Application {
		if !d.Application[a].Ok() {
			return false
		}
	}
	return true
}

func (i Infrastructure) Ok() bool {
	return i.Success
}
