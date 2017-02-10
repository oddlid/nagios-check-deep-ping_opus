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

type Keys []string

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

func (k Keys) MaxLen() int {
	max := 0
	for i := range k {
		klen := len(k[i])
		if klen > max {
			max = klen
		}
	}
	return max
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

// _hdr is a simplified version of _pp that is used for printing headers
func _hdr(w io.Writer, prefix, key, sep string, level int) {
	fmt.Fprintf(w, fmt.Sprintf("%s%s %s\n", strings.Repeat(prefix, level), key, sep))
}

// _keys is a little helper function that returns a Keys{} instance and the max key length
// just to save some typing
func _keys(ks ...string) (int, Keys) {
	klen := len(ks)
	k := make(Keys, klen)
	for i := range ks {
		k[i] = ks[i]
	}
	return k.MaxLen(), k
}

func (i Infrastructure) pp(w io.Writer, prefix string, level int) {
	max, k := _keys(T_NA, T_TY, T_SU, T_RT)
	p := func(k, v string) {
		_pp(w, prefix, k, v, max, level)
	}
	p(k[0], i.Name)
	p(k[1], i.Type)
	p(k[2], fmt.Sprintf("%t", i.Success))
	p(k[3], fmt.Sprintf("%f", i.ResponseTime))
}

func (a Application) pp(w io.Writer, prefix string, level int) {
	// Fix for Görans OCD:
	// We need to build the keys slice dynamically depending on which keys have values.
	// Then we only use the maxlength from that, and can NOT use the array otherwise,
	// because we can never know at which index the keys are
	k := Keys{T_SU, T_SK, T_RT, T_DE} // non-string props can be added regardless
	_k := func(k1, k2 string) {
		if k1 != "" {
			k = append(k, k2)
		}
	}

	_k(a.LongName, T_LN)
	_k(a.ShortName, T_SN)
	_k(a.Version, T_VS)
	_k(a.FailureReason, T_FR)
	_k(a.EndPoint, T_EP)

	max := k.MaxLen()
	k = nil // not needed after this

	p := func(k, v string) {
		if v != "" {
			_pp(w, prefix, k, v, max, level)
		}
	}

	p(T_LN, a.LongName)
	p(T_SN, a.ShortName)
	p(T_VS, a.Version)
	p(T_FR, a.FailureReason)
	p(T_EP, a.EndPoint)
	p(T_SU, fmt.Sprintf("%t", a.Success))
	p(T_SK, fmt.Sprintf("%t", a.Skipped))
	p(T_RT, fmt.Sprintf("%f", a.ResponseTime))

	if a.Dependencies != nil {
		deplen := len(a.Dependencies)
		for i := range a.Dependencies {
			_hdr(w, prefix, fmt.Sprintf("%s (#%d/%d)", T_DE, i+1, deplen), "=>", level)
			a.Dependencies[i].pp(w, prefix, level+1)
		}
	}
}

func (d Dependencies) pp(w io.Writer, prefix string, level int) {
	if d.Infrastructure != nil {
		inflen := len(d.Infrastructure)
		for i := range d.Infrastructure {
			_hdr(w, prefix, fmt.Sprintf("%s (#%d/%d)", T_IS, i+1, inflen), "=>", level)
			d.Infrastructure[i].pp(w, prefix, level+1)
		}
	}
	if d.Application != nil {
		applen := len(d.Application)
		for j := range d.Application {
			_hdr(w, prefix, fmt.Sprintf("%s (#%d/%d)", T_AP, j+1, applen), "=>", level)
			d.Application[j].pp(w, prefix, level+1)
		}
	}
}

func (pr PingResponse) pp(w io.Writer, prefix string, level int) {
	max, k := _keys("URL", T_HC, T_RT, "Error", T_AP)
	p := func(k, v string) {
		_pp(w, prefix, k, v, max, level)
	}
	fmt.Fprintf(w, "===== BEGIN: %s =====\n", T_PR)
	p(k[0], pr.URL)
	p(k[1], fmt.Sprintf("%d", pr.HTTPCode))
	p(k[2], fmt.Sprintf("%f", pr.ResponseTime))
	if pr.Err != nil {
		p(k[3], pr.Err.Error())
	}
	if pr.Application != nil {
		applen := len(pr.Application)
		for i := range pr.Application {
			_hdr(w, prefix, fmt.Sprintf("%s (#%d/%d)", k[4], i+1, applen), "=>", level)
			pr.Application[i].pp(w, prefix, level+1)
		}
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
