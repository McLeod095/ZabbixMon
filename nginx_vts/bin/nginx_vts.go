// Build for Linux
// GOOS=linux GOARCH=amd64 go build -ldflags "-s -w"
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type NginxVts struct {
	NginxVersion string `json:"nginxVersion"`
	LoadMsec     int64  `json:"loadMsec"`
	NowMsec      int64  `json:"nowMsec"`
	Connections  struct {
		Active   int `json:"active"`
		Reading  int `json:"reading"`
		Writing  int `json:"writing"`
		Waiting  int `json:"waiting"`
		Accepted int `json:"accepted"`
		Handled  int `json:"handled"`
		Requests int `json:"requests"`
	} `json:"connections"`
	ServerZones   map[string]Server     `json:"serverZones"`
	UpstreamZones map[string][]Upstream `json:"upstreamZones"`
	CacheZones    map[string]Cache      `json:"cacheZones"`
}

type Server struct {
	RequestCounter int `json:"requestCounter"`
	InBytes        int `json:"inBytes"`
	OutBytes       int `json:"outBytes"`
	Responses      struct {
		OneXx       int `json:"1xx"`
		TwoXx       int `json:"2xx"`
		ThreeXx     int `json:"3xx"`
		FourXx      int `json:"4xx"`
		FiveXx      int `json:"5xx"`
		Miss        int `json:"miss"`
		Bypass      int `json:"bypass"`
		Expired     int `json:"expired"`
		Stale       int `json:"stale"`
		Updating    int `json:"updating"`
		Revalidated int `json:"revalidated"`
		Hit         int `json:"hit"`
		Scarce      int `json:"scarce"`
	} `json:"responses"`
	OverCounts struct {
		MaxIntegerSize float64 `json:"maxIntegerSize"`
		RequestCounter int     `json:"requestCounter"`
		InBytes        int     `json:"inBytes"`
		OutBytes       int     `json:"outBytes"`
		OneXx          int     `json:"1xx"`
		TwoXx          int     `json:"2xx"`
		ThreeXx        int     `json:"3xx"`
		FourXx         int     `json:"4xx"`
		FiveXx         int     `json:"5xx"`
		Miss           int     `json:"miss"`
		Bypass         int     `json:"bypass"`
		Expired        int     `json:"expired"`
		Stale          int     `json:"stale"`
		Updating       int     `json:"updating"`
		Revalidated    int     `json:"revalidated"`
		Hit            int     `json:"hit"`
		Scarce         int     `json:"scarce"`
	} `json:"overCounts"`
}

type Upstream struct {
	Server         string `json:"server"`
	RequestCounter int    `json:"requestCounter"`
	InBytes        int    `json:"inBytes"`
	OutBytes       int    `json:"outBytes"`
	Responses      struct {
		OneXx   int `json:"1xx"`
		TwoXx   int `json:"2xx"`
		ThreeXx int `json:"3xx"`
		FourXx  int `json:"4xx"`
		FiveXx  int `json:"5xx"`
	} `json:"responses"`
	ResponseMsec int  `json:"responseMsec"`
	Weight       int  `json:"weight"`
	MaxFails     int  `json:"maxFails"`
	FailTimeout  int  `json:"failTimeout"`
	Backup       bool `json:"backup"`
	Down         bool `json:"down"`
	OverCounts   struct {
		MaxIntegerSize float64 `json:"maxIntegerSize"`
		RequestCounter int     `json:"requestCounter"`
		InBytes        int     `json:"inBytes"`
		OutBytes       int     `json:"outBytes"`
		OneXx          int     `json:"1xx"`
		TwoXx          int     `json:"2xx"`
		ThreeXx        int     `json:"3xx"`
		FourXx         int     `json:"4xx"`
		FiveXx         int     `json:"5xx"`
	} `json:"overCounts"`
}

type Cache struct {
	MaxSize   int `json:"maxSize"`
	UsedSize  int `json:"usedSize"`
	InBytes   int `json:"inBytes"`
	OutBytes  int `json:"outBytes"`
	Responses struct {
		Miss        int `json:"miss"`
		Bypass      int `json:"bypass"`
		Expired     int `json:"expired"`
		Stale       int `json:"stale"`
		Updating    int `json:"updating"`
		Revalidated int `json:"revalidated"`
		Hit         int `json:"hit"`
		Scarce      int `json:"scarce"`
	} `json:"responses"`
	OverCounts struct {
		MaxIntegerSize float64 `json:"maxIntegerSize"`
		InBytes        int     `json:"inBytes"`
		OutBytes       int     `json:"outBytes"`
		Miss           int     `json:"miss"`
		Bypass         int     `json:"bypass"`
		Expired        int     `json:"expired"`
		Stale          int     `json:"stale"`
		Updating       int     `json:"updating"`
		Revalidated    int     `json:"revalidated"`
		Hit            int     `json:"hit"`
		Scarce         int     `json:"scarce"`
	} `json:"overCounts"`
}

type LLD struct {
	Data []map[string]string `json:"data"`
}

var uri string

func init() {
	flag.StringVar(&uri, "uri", "http://localhost:8899/status/format/json", "uri for nginx_vts page")
	flag.Parse()
}

func main() {
	http.DefaultClient.Timeout = 10 * time.Second
	resp, err := http.DefaultClient.Get(uri)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		log.Fatalln(resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var nginxVtx NginxVts
	err = json.Unmarshal(data, &nginxVtx)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("- nginx.status[connections,active]", nginxVtx.Connections.Active)
	fmt.Println("- nginx.status[connections,reading]", nginxVtx.Connections.Reading)
	fmt.Println("- nginx.status[connections,waiting]", nginxVtx.Connections.Waiting)
	fmt.Println("- nginx.status[connections,writing]", nginxVtx.Connections.Writing)
	fmt.Println("- nginx.status[connections,accepted]", nginxVtx.Connections.Accepted)
	fmt.Println("- nginx.status[connections,handled]", nginxVtx.Connections.Handled)
	fmt.Println("- nginx.status[connections,requests]", nginxVtx.Connections.Requests)

	var lld LLD
	var databuffer string
	for host, s := range nginxVtx.ServerZones {
		if host == "*" {
			host = "all"
		}
		lld.Data = append(lld.Data, map[string]string{"{#SERVERNAME}": host})
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",requests] %d\n", host, s.RequestCounter)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,1xx] %d\n", host, s.Responses.OneXx)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,2xx] %d\n", host, s.Responses.TwoXx)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,3xx] %d\n", host, s.Responses.ThreeXx)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,4xx] %d\n", host, s.Responses.FourXx)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,5xx] %d\n", host, s.Responses.FiveXx)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,bypass] %d\n", host, s.Responses.Bypass)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,expired] %d\n", host, s.Responses.Expired)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,hit] %d\n", host, s.Responses.Hit)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,miss] %d\n", host, s.Responses.Miss)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,revalidated] %d\n", host, s.Responses.Revalidated)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,scarce] %d\n", host, s.Responses.Scarce)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,stale] %d\n", host, s.Responses.Stale)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",response,updating] %d\n", host, s.Responses.Updating)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",in] %d\n", host, s.InBytes)
		databuffer += fmt.Sprintf("- nginx.status[\"%s\",out] %d\n", host, s.OutBytes)
	}

	if len(lld.Data) > 0 {
		b, err := json.Marshal(lld)
		if err == nil {
			fmt.Printf("- nginx.server.discovery %s\n", b)
		}
	}
	lld = LLD{}

	for name, upstreamList := range nginxVtx.UpstreamZones {
		var total, one, two, three, four, five, inbytes, outbytes int
		for _, s := range upstreamList {
			total += s.RequestCounter
			two += s.Responses.TwoXx
			one += s.Responses.OneXx
			three += s.Responses.ThreeXx
			four += s.Responses.FourXx
			five += s.Responses.FiveXx

			inbytes += s.InBytes
			outbytes += s.OutBytes
		}
		lld.Data = append(lld.Data, map[string]string{"{#UPSTREAM}": name})
		databuffer += fmt.Sprintf("- nginx.status[upstream,\"%s\",total] %d\n", name, total)
		databuffer += fmt.Sprintf("- nginx.status[upstream,\"%s\",1xx] %d\n", name, one)
		databuffer += fmt.Sprintf("- nginx.status[upstream,\"%s\",2xx] %d\n", name, two)
		databuffer += fmt.Sprintf("- nginx.status[upstream,\"%s\",3xx] %d\n", name, three)
		databuffer += fmt.Sprintf("- nginx.status[upstream,\"%s\",4xx] %d\n", name, four)
		databuffer += fmt.Sprintf("- nginx.status[upstream,\"%s\",5xx] %d\n", name, five)
		databuffer += fmt.Sprintf("- nginx.status[upstream,\"%s\",in] %d\n", name, inbytes)
		databuffer += fmt.Sprintf("- nginx.status[upstream,\"%s\",out] %d\n", name, outbytes)
	}
	if len(lld.Data) > 0 {
		b, err := json.Marshal(lld)
		if err == nil {
			fmt.Printf("- nginx.upstream.discovery %s\n", b)
		}
	}

	lld = LLD{}

	for zone, s := range nginxVtx.CacheZones {
		lld.Data = append(lld.Data, map[string]string{"{#CACHEZONE}": zone})
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",bypass] %d\n", zone, s.Responses.Bypass)
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",expired] %d\n", zone, s.Responses.Expired)
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",hit] %d\n", zone, s.Responses.Hit)
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",miss] %d\n", zone, s.Responses.Miss)
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",revalidated] %d\n", zone, s.Responses.Revalidated)
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",scarce] %d\n", zone, s.Responses.Scarce)
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",stale] %d\n", zone, s.Responses.Stale)
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",updating] %d\n", zone, s.Responses.Updating)
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",in] %d\n", zone, s.InBytes)
		databuffer += fmt.Sprintf("- nginx.status[cachezone,\"%s\",out] %d\n", zone, s.OutBytes)
	}

	if len(lld.Data) > 0 {
		b, err := json.Marshal(lld)
		if err == nil {
			fmt.Printf("- nginx.cachezone.discovery %s\n", b)
		}
	}
	fmt.Print(databuffer)
}
