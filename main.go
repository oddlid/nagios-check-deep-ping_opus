/*
Deep Ping check for Opus/LovelyBridge

Odd E. Ebbesen, 2015-10-05 09:03:07
*/

package main

import (
	"crypto/tls"
	"fmt"
	//"encoding/json"
	"github.com/PuerkitoBio/goquery"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"net/http"
	"os"
	"strings"
	"strconv"
	"time"
)

const (
	VERSION    string  = "2016-01-08"
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

type ResponseStatus struct {
	Success      bool
	ResponseTime time.Duration
}

type Infrastructure struct {
	Name   string
	Type   string
	Status ResponseStatus
}

type DPApp struct {
	LongName     string
	ShortName    string
	Version      string
	Status       ResponseStatus
	Dependencies []Infrastructure
}

type PingResponse struct {
	HTTPCode int
	RTime    time.Duration
	App      DPApp
}

func (da *DPApp) AddDep(i *Infrastructure) *DPApp {
	da.Dependencies = append(da.Dependencies, *i)
	return da
}

func (pr PingResponse) String() string {
	str := fmt.Sprintf("Application\n\tlong-name    : %q\n\tshort-name   : %q\n\tversion      : %q\n\tsuccess      : %t\n\tresponsetime : %f\n",
		pr.App.LongName, pr.App.ShortName, pr.App.Version, pr.App.Status.Success, pr.App.Status.ResponseTime.Seconds())
	for i, val := range pr.App.Dependencies {
		str += fmt.Sprintf("Infrastructure (#%d)\n\ttype         : %q\n\tname         : %q\n\tsuccess      : %t\n\tresponsetime : %f\n",
			i, val.Type, val.Name, val.Status.Success, val.Status.ResponseTime.Seconds())
	}
	return str
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

func scrape(url string, chRes chan PingResponse, chCtrl chan bool) {
	defer func() {
		chCtrl <- true // send signal that parsing/scraping is done
	}()

	mstr := []string{ // match strings
		"application",
		"long-name",
		"short-name",
		"version",
		"success",
		"true",
		"responsetime",
		"dependencies",
		"infrastructure",
		"type",
		"name",
	}

	// little helper for stuff I do more than once
	// checks that tag <success> contains "true", as in: <success>true</success>
	success := func(s *goquery.Selection) bool {
		return s.Find(mstr[4]).First().Text() == mstr[5]
	}
	// one more helper, parses out number from <responsetime>01234..</responsetime>
	responsetime := func(s *goquery.Selection) (int, error) {
		rt, err := strconv.Atoi(s.Find(mstr[6]).First().Text())
		if err != nil {
			log.Errorf("Unable to parse responsetime: %s", err)
			return 0, err
		}
		return rt, nil
	}

	t_start := time.Now() // start timer for request
	resp, err := geturl(url)
	t_end := time.Now() // end timer
	if err != nil {
		log.Debugf("Unable to fetch URL: %q, error: %s", url, err)
		nagios_result(E_CRITICAL, S_CRITICAL, "Unable to fetch URL:", url, time.Duration(t_end.Sub(t_start)).Seconds(), 0, 0, &PingResponse{})
	}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.Fatalf("Problem loading document, error: %s", err)
	}

	pr := PingResponse{HTTPCode: resp.StatusCode, RTime: t_end.Sub(t_start), App: DPApp{}}

	appl := doc.Find(mstr[0])            // <application>...</application>
	appLN, foundLN := appl.Attr(mstr[1]) // <... long-name="...">
	if foundLN {
		pr.App.LongName = appLN
	}
	appSN, foundSN := appl.Attr(mstr[2]) // <... short-name="...">
	if foundSN {
		pr.App.ShortName = appSN
	}
	appVer, foundVer := appl.Attr(mstr[3]) // <... version="...">
	if foundVer {
		pr.App.Version = appVer
	}

	rs := ResponseStatus{}
	rs.Success = success(appl)
	rt, err := responsetime(appl)
	if err == nil {
		rs.ResponseTime = time.Duration(rt) * time.Second
		pr.App.Status = rs
	}
	// <dependencies>...</dependencies>
	appl.Find(mstr[7]).Find(mstr[8]).Each(func(i int, sel *goquery.Selection) {
		_inf := Infrastructure{}
		_type, _found := sel.Attr(mstr[9]) // <... type="...">
		if _found {
			_inf.Type = _type
		}
		_name, _found := sel.Attr(mstr[10]) // <... name="...">
		if _found {
			_inf.Name = _name
		}
		_inf.Status.Success = success(sel)
		_rt, _err := responsetime(sel)
		if _err == nil {
			_inf.Status.ResponseTime = time.Duration(_rt) * time.Second
			pr.App.AddDep(&_inf)
		}
	})

	// send result
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
	chCtrl := make(chan bool)
	defer close(chPRes)
	defer close(chCtrl)

	// run in parallell thread
	go scrape(dpurl, chPRes, chCtrl)

	select {
	case res := <-chPRes:
		log.Debugf("Response object: %#v", res)

		/* This was also a good way to see objects
		jres, err := json.Marshal(res)
		if err != nil {
			log.Fatalf("Error converting to JSON: %s\n", err)
		}
		//log.Debugf("Response JSON: %s\n", string(jres))
		fmt.Printf("%s\n", string(jres))
		*/

		if res.HTTPCode != 200 {
			log.Warnf("HTTP: %d (%s) - please do the needful", res.HTTPCode, dpurl)
			msg := fmt.Sprintf("Unexpected HTTP return code: %d", res.HTTPCode)
			nagios_result(E_CRITICAL, S_CRITICAL, msg, path, res.RTime.Seconds(), warn, crit, &res)
		}
		if !res.App.Status.Success {
			log.Warnf("No success. Sooo needful... Buhu...")
			msg := fmt.Sprintf("Response tagged as unsuccessful (appl: %q, version: %q)", res.App.LongName, res.App.Version)
			nagios_result(E_CRITICAL, S_CRITICAL, msg, path, res.RTime.Seconds(), warn, crit, &res)
		}
		if res.RTime.Seconds() >= crit {
			msg := fmt.Sprintf("Too long response time (>= %ds)", int(crit))
			nagios_result(E_CRITICAL, S_CRITICAL, msg, path, res.RTime.Seconds(), warn, crit, &res)
		}
		if res.RTime.Seconds() >= warn {
			msg := fmt.Sprintf("Too long response time (>= %ds)", int(warn))
			nagios_result(E_WARNING, S_WARNING, msg, path, res.RTime.Seconds(), warn, crit, &res)
		}

		msg := fmt.Sprintf("Looking good")
		nagios_result(E_OK, S_OK, msg, path, res.RTime.Seconds(), warn, crit, &res)

	case <-chCtrl: // not really meaningful when not looping over worker goroutines
		log.Info("Got done signal on control channel. Bye.")
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
