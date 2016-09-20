/*
Deep Ping check for Opus/LovelyBridge

New version that parses XML more "for real", as the previous attempt
started failing due to ever changing output from the DPs

Odd E. Ebbesen, 2016-04-17 22:36:09

*/

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	VERSION    string  = "2016-09-20"
	UA         string  = "VGT Deep Pings/3.0"
	defPort    int     = 80
	defWarn    float64 = 10.0
	defCrit    float64 = 15.0
	defTmout   float64 = 30.0
	defProt    string  = "http"
	S_OK       string  = "OK"
	S_WARNING  string  = "WARNING"
	S_CRITICAL string  = "CRITICAL"
	S_UNKNOWN  string  = "UNKNOWN"
	E_OK       int     = 0
	E_WARNING  int     = 1
	E_CRITICAL int     = 2
	E_UNKNOWN  int     = 3
)

// if true, we include long output from the check
var verbose bool = false

// Trying to simplify debugging
func _debug(f func()) {
	lvl := log.GetLevel()
	if lvl == log.DebugLevel {
		f()
	}
}

func nagios_result(ecode int, status, desc, path string, rtime, warn, crit float64, pr *PingResponse) {
	msg := fmt.Sprintf("%s: % s, Path: %q, Response time: %f|time=%fs;%f;%f\n",
		status, desc, path, rtime, rtime, warn, crit)
	if verbose {
		msg += pr.String()
		log.Debugf("pp_count : %d\n", pp_count)
	}
	fmt.Println(msg)
	os.Exit(ecode)
}

func geturl(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", UA)

	tr := &http.Transport{DisableKeepAlives: true} // we're not reusing the connection, so don't let it hang open
	if strings.Index(url, "https") >= 0 {
		// Verifying certs is not the job of this plugin,
		// so we save ourselves a lot of grief by skipping any SSL verification
		// Could be a good idea for later to set this at runtime instead
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Transport: tr}

	return client.Do(req)
}

func scrape(url string, chRes chan PingResponse) {
	t_start := time.Now()
	resp, err := geturl(url)
	pr := PingResponse{
		ResponseTime: time.Duration(time.Now().Sub(t_start)).Seconds(),
		URL:          url,
	}
	if err != nil {
		log.Error(err)
		pr.Err = err
		nagios_result(
			E_CRITICAL,
			S_CRITICAL,
			fmt.Sprintf("Unable to fetch URL: %q", url),
			"",
			pr.ResponseTime,
			0,
			0,
			&pr)
	}
	defer resp.Body.Close()

	pr.HTTPCode = resp.StatusCode
	if pr.HTTPCode != http.StatusOK {
		chRes <- pr
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		pr.Err = err
		nagios_result(
			E_CRITICAL,
			S_CRITICAL,
			fmt.Sprintf("Error reading response body from %q", url),
			"",
			pr.ResponseTime,
			0,
			0,
			&pr)
	}
	err = xml.Unmarshal(data, &pr)
	if err != nil {
		log.Error(err)
		log.Debugf("Response body:\n%s", data)
		pr.Err = err
		nagios_result(
			E_UNKNOWN,
			S_UNKNOWN,
			fmt.Sprintf("Unable to parse returned (XML) content from %q", url),
			"",
			pr.ResponseTime,
			0,
			0,
			&pr)
	}

	chRes <- pr
}

func run_check(c *cli.Context) {
	furl := c.String("url") // f for full-url
	prot := c.String("protocol")
	host := c.String("hostname")
	port := c.Int("port")
	path := c.String("urlpath")
	warn := c.Float64("warning")
	crit := c.Float64("critical")
	tmout := c.Float64("timeout")

	var dpurl string
	if furl != "" {
		dpurl = furl
		tmpurl, err := url.Parse(furl)
		if err == nil {
			path = tmpurl.EscapedPath()
			//log.Debugf("URL Path : %s", path)
		}
	} else {
		dpurl = fmt.Sprintf("%s://%s:%d%s", prot, host, port, path)
	}

	_debug(func() {
		log.Debugf("URL:     : %q", furl)
		log.Debugf("Protocol : %s", prot)
		log.Debugf("Host     : %s", host)
		log.Debugf("Port     : %d", port)
		log.Debugf("UPath    : %s", path)
		log.Debugf("Warning  : %f", warn)
		log.Debugf("Critical : %f", crit)
		log.Debugf("Timeout  : %f", tmout)
		log.Debugf("DP URL   : %s", dpurl)
	})

	chPRes := make(chan PingResponse)
	defer close(chPRes)

	// run in parallell thread
	go scrape(dpurl, chPRes)

	select {
	case res := <-chPRes:
		if res.HTTPCode != http.StatusOK {
			msg := fmt.Sprintf("Unexpected HTTP response code: %d", res.HTTPCode)
			nagios_result(E_CRITICAL, S_CRITICAL, msg, path, res.ResponseTime, warn, crit, &res)
		}
		if !res.Ok() {
			_debug(func() {
				var buf bytes.Buffer
				written, err := res.DumpJSON(&buf, true)
				if err != nil {
					log.Error(err)
				}
				log.Debugf("XML as JSON (%d bytes):\n%s", written, buf.String())
			})
			msg := "Response tagged as unsuccessful, see long output for details"
			nagios_result(E_CRITICAL, S_CRITICAL, msg, path, res.ResponseTime, warn, crit, &res)
		}
		if res.ResponseTime >= crit {
			msg := fmt.Sprintf("Response time above critical [ %ds ] limit", int(crit))
			nagios_result(E_CRITICAL, S_CRITICAL, msg, path, res.ResponseTime, warn, crit, &res)
		}
		if res.ResponseTime >= warn {
			msg := fmt.Sprintf("Response time above warning [ %ds ] limit", int(warn))
			nagios_result(E_WARNING, S_WARNING, msg, path, res.ResponseTime, warn, crit, &res)
		}
		// Got here, all good
		nagios_result(E_OK, S_OK, "Looking good", path, res.ResponseTime, warn, crit, &res)

	case <-time.After(time.Second * time.Duration(tmout)):
		fmt.Printf("%s: DP %q timed out after %d seconds.\n", S_CRITICAL, dpurl, int(tmout))
		os.Exit(E_CRITICAL)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "check_deep_ping_opus"
	app.Version = VERSION
	app.Author = "Odd E. Ebbesen"
	app.Email = "odd.ebbesen@wirelesscar.com"
	app.Usage = "XML Rest API parser for WirelessCar Deep Pings, Opus version"
	//app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "url, U",
			Usage: "Full URL to check, in the format: protocol://(hostname|IP)(?:port)/path",
		},
		cli.StringFlag{
			Name:  "hostname, H",
			Value: "localhost",
			Usage: "Hostname or IP to check",
		},
		cli.IntFlag{
			Name:  "port, p",
			Value: defPort,
			Usage: "TCP port",
		},
		cli.StringFlag{
			Name:  "protocol, P",
			Value: defProt,
			Usage: "Protocol to use (http or https)",
		},
		cli.StringFlag{
			Name:  "urlpath, u",
			Usage: "The path part of the url",
		},
		cli.Float64Flag{
			Name:  "warning, w",
			Value: defWarn,
			Usage: "Response time to result in WARNING status, in seconds",
		},
		cli.Float64Flag{
			Name:  "critical, c",
			Value: defCrit,
			Usage: "Response time to result in CRITICAL status, in seconds",
		},
		cli.Float64Flag{
			Name:  "timeout, t",
			Value: defTmout,
			Usage: "Number of seconds before connection times out",
		},
		cli.StringFlag{
			Name:  "log-level, l",
			Value: "fatal",
			Usage: "Log level (options: debug, info, warn, error, fatal, panic)",
		},
		cli.BoolFlag{
			Name:        "verbose, V",
			Usage:       "Verbose output. Includes a full dump of the returned data",
			Destination: &verbose,
			EnvVar:      "OPUS_DP_VERBOSE",
		},
		cli.BoolFlag{
			Name:   "debug, d",
			Usage:  "Run in debug mode",
			EnvVar: "OPUS_DP_DEBUG",
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetOutput(os.Stdout)
		level, err := log.ParseLevel(c.String("log-level"))
		if err != nil {
			log.Fatal(err.Error())
		}
		log.SetLevel(level)
		if !c.IsSet("log-level") && !c.IsSet("l") && c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	}

	app.Action = run_check
	app.Run(os.Args)
}
