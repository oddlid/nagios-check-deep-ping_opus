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
	T_DP string = "\t"							// Default Prefix, for pretty printing
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

type PingResponse struct {
	XMLName      xml.Name      `xml:"PingResponse" json:",omitempty"`
	Application  []Application `xml:"application,omitempty" json:",omitempty"`
	Err          error         `json:"error,omitempty"`
	ResponseTime float64       `json:",omitempty"`
	PP_Prefix    string        `json:",omitempty"`
	HTTPCode     int           `json:",omitempty"`
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
}

func (i Infrastructure) pp(w io.Writer, prefix string, level int) {
	wi := 12
	_pp(w, prefix, T_NA, i.Name, wi, level)
	_pp(w, prefix, T_TY, i.Type, wi, level)
	_pp(w, prefix, T_SU, fmt.Sprintf("%t", i.Success), wi, level)
	_pp(w, prefix, T_RT, fmt.Sprintf("%v", i.ResponseTime), wi, level)
}

func (a Application) pp(w io.Writer, prefix string, level int) {
	wi := 13
	_pp(w, prefix, T_LN, a.LongName, wi, level)
	_pp(w, prefix, T_SN, a.ShortName, wi, level)
	_pp(w, prefix, T_VS, a.Version, wi, level)
	_pp(w, prefix, T_FR, a.FailureReason, wi, level)
	_pp(w, prefix, T_EP, a.EndPoint, wi, level)
	_pp(w, prefix, T_SU, fmt.Sprintf("%t", a.Success), wi, level)
	_pp(w, prefix, T_SK, fmt.Sprintf("%t", a.Skipped), wi, level)
	_pp(w, prefix, T_RT, fmt.Sprintf("%f", a.ResponseTime), wi, level)

	if len(a.Dependencies) > 0 {
		for i := range a.Dependencies {
			_pp(w, prefix, T_DE, "", wi, level)
			a.Dependencies[i].pp(w, prefix, level+1)
		}
	}
}

func (d Dependencies) pp(w io.Writer, prefix string, level int) {
	wi := 14
	if len(d.Infrastructure) > 0 {
		for i := range d.Infrastructure {
			_pp(w, prefix, T_IS, "", wi, level)
			d.Infrastructure[i].pp(w, prefix, level+1)
		}
	}
	if len(d.Application) > 0 {
		for j := range d.Application {
			_pp(w, prefix, T_AP, "", wi, level)
			d.Application[j].pp(w, prefix, level+1)
		}
	}
}

func (p PingResponse) pp(w io.Writer, prefix string, level int) {
	wi := 12
	fmt.Fprintf(w, "=== BEGIN: %s ===\n", T_PR)
	_pp(w, prefix, T_HC, fmt.Sprintf("%d", p.HTTPCode), wi, level)
	_pp(w, prefix, T_RT, fmt.Sprintf("%f", p.ResponseTime), wi, level)
	_pp(w, prefix, "Error", fmt.Sprintf("%v", p.Err), wi, level)
	for i := range p.Application {
		_pp(w, prefix, T_AP, "", wi, level)
		p.Application[i].pp(w, prefix, level+1)
	}
	fmt.Fprintf(w, "=== END: %s ===\n", T_PR)
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
