/*
Deep Ping check for Opus/LovelyBridge

New version that parses XML more "for real", as the previous attempt
started failing due to ever changing output from the DPs

Odd E. Ebbesen, 2016-04-17 22:36:09

*/

package main

import (
	"crypto/tls"
	//"encoding/json"
	"encoding/xml"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	VERSION    string  = "2016-04-19"
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

// Trying to simplify debugging
func _debug(f func()) {
	lvl := log.GetLevel()
	if lvl == log.DebugLevel {
		f()
	}
}

func nagios_result(ecode int, status, desc, path string, rtime, warn, crit float64, pr *PingResponse) {
	// not sure if we need the multiline output or not. Might drop it.
	fmt.Printf("%s: %s, %q, response time: %f|time=%fs;%fs;%fs\n%s",
		status, desc, path, rtime, rtime, warn, crit, pr.String())
	os.Exit(ecode)
}

func geturl(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", UA)

	tr := &http.Transport{DisableKeepAlives: true} // we're not reusing the connection, so don't let it hang open
	if strings.Index(url, "https") >= 0 {
		// Verifying certs is not the job of this plugin, so we save ourselves a lot of grief 
		// by skipping any SSL verification
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Transport: tr}

	return client.Do(req)
}

func scrape(url string, chRes chan PingResponse) {
	t_start := time.Now()
	resp, err := geturl(url)
	pr := PingResponse{ResponseTime: time.Duration(time.Now().Sub(t_start)).Seconds(), URL: url}
	if err != nil {
		pr.Err = err
		nagios_result(E_CRITICAL, S_CRITICAL, "Unable to fetch URL:", url, pr.ResponseTime, 0, 0, &pr)
	}
	pr.HTTPCode = resp.StatusCode
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		pr.Err = err
		nagios_result(E_CRITICAL, S_CRITICAL, "Error reading response body", url, pr.ResponseTime, 0, 0, &pr)
	}
	err = xml.Unmarshal(data, &pr)
	if err != nil {
		pr.Err = err
		nagios_result(E_CRITICAL, S_CRITICAL, "Unable to parse returned (XML) content", url, pr.ResponseTime, 0, 0, &pr)
	}

	// The not so lightweight JSON processing here is only actually run
	// if the log level is at "debug"
	// switch this to pr.DumpJSON() instead if enabled again
	//_debug(func() {
	//	jbytes, err := json.MarshalIndent(pr, "", " ")
	//	if err != nil {
	//		log.Error(err)
	//	}
	//	log.Debugf("XML as JSON:\n%s", jbytes)
	//})

	chRes <- pr
}

func run_check(c *cli.Context) {
	prot := c.String("protocol")
	host := c.String("hostname")
	port := c.Int("port")
	path := c.String("urlpath")
	warn := c.Float64("warning")
	crit := c.Float64("critical")
	tmout := c.Float64("timeout")

	log.Debugf("Protocol : %s", prot)
	log.Debugf("Host     : %s", host)
	log.Debugf("Port     : %d", port)
	log.Debugf("UPath    : %s", path)
	log.Debugf("Warning  : %f", warn)
	log.Debugf("Critical : %f", crit)
	log.Debugf("Timeout  : %f", tmout)

	dpurl := fmt.Sprintf("%s://%s:%d%s", prot, host, port, path)
	log.Debugf("DP URL   : %s", dpurl)

	chPRes := make(chan PingResponse)
	defer close(chPRes)

	// run in parallell thread
	go scrape(dpurl, chPRes)

	select {
	case res := <-chPRes:
		log.Debugf("Response object:\n%#v", res)
		log.Debug("\n", res.String())
		log.Debugf("Overall success: %t\n", res.Success())

		if res.HTTPCode != 200 {
			// CRIT
		}
	case <-time.After(time.Second * time.Duration(tmout)):
		//log.Errorf("%s: DP %q timed out after %d seconds", S_CRITICAL, dpurl, int(tmout))
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
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
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
			Name:   "debug, d",
			Usage:  "Run in debug mode",
			EnvVar: "DEBUG",
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
