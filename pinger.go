package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	tunnel "github.com/dariopb/goreverselb/pkg"
	"github.com/labstack/echo"
)

type CommandObj struct {
	Target string `json:"target,omitempty"`
	Text   string `json:"text,omitempty"`
}

type ResponseObj struct {
	ErrorCode     int    `json:"errorCode"`
	HTTPErrorCode int    `json:"httpErrorCode,omitempty"`
	Message       string `json:"message,omitempty"`
}

func main() {
	portstr := os.Getenv("PORT")
	port, err := strconv.Atoi(portstr)
	if err != nil {
		port = 8080
	}

	fmt.Printf("starting pinger on [%d]...\n", port)

	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			fmt.Println(addr)
		}
	}

	tunnelEndpointAPI := os.Getenv("LB_API_ENDPOINT")
	servicename := os.Getenv("LB_SERVICENAME")

	if tunnelEndpointAPI != "" && servicename != "" {
		token := os.Getenv("LB_TOKEN")
		td := tunnel.TunnelData{
			ServiceName:          servicename,
			BackendAcceptBacklog: 1,
			FrontendData: tunnel.FrontendData{
				Port: 0,
			},
			TargetPort:      port,
			TargetAddresses: []string{"127.0.0.1"},
			Token:           token,
		}

		_, err = tunnel.NewMuxTunnelClient(tunnelEndpointAPI, td)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("Not starting lb association")
	}

	e := echo.New()
	e.HideBanner = true

	e.Static("/xterm", "web/index.html")
	e.Static("/web", "web")
	e.Static("/", "web")

	// Routes
	e.GET("/", hello)
	e.GET("/proxy", proxy)
	e.GET("/resolveName", resolveDns)
	e.GET("/xterm/ws", xterm)

	// Start server
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", port)))
}

func hello(c echo.Context) error {
	hostname, _ := os.Hostname()
	banner := fmt.Sprintf("I'm alive on [%s] (%s on %s/%s). Available routes: ",
		hostname, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	routes := c.Echo().Routes()
	for _, route := range routes {
		banner = banner + route.Path + " "
	}

	return c.JSON(http.StatusOK, banner)
}

func resolveDns(c echo.Context) error {
	resp := ResponseObj{
		ErrorCode: -1,
	}

	cmd := new(CommandObj)
	err := c.Bind(cmd)
	if err != nil {
		resp.Message = fmt.Sprintf("Error binding: [%s]", err)
		return c.JSON(http.StatusOK, resp)
	}

	ips, err := net.LookupIP(cmd.Target)
	if err != nil {
		resp.Message = fmt.Sprintf("Could not resolve name: [%s]: [%s]", cmd.Target, err)
	}
	for _, ip := range ips {
		resp.Message = resp.Message + ip.String() + " "
	}

	resp.ErrorCode = 0
	return c.JSON(http.StatusOK, resp)
}

func proxy(c echo.Context) error {
	httpClient := http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   4 * time.Second,
				KeepAlive: 4 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   4 * time.Second,
			ResponseHeaderTimeout: 4 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 5 * time.Second,
	}
	resp := ResponseObj{
		ErrorCode: -1,
	}

	cmd := new(CommandObj)
	err := c.Bind(cmd)
	if err != nil {
		resp.Message = err.Error()
		return c.JSON(http.StatusOK, resp)
	}

	// invoke it
	response, err := httpClient.Get(cmd.Target)
	if err != nil {
		resp.Message = fmt.Sprintf("HTTP request failed with error: [%s]", err)
		return c.JSON(http.StatusOK, resp)
	}

	resp.HTTPErrorCode = response.StatusCode
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		resp.Message = fmt.Sprintf("HTTP request failed retrieving data: [%s]", err)
		return c.JSON(http.StatusOK, resp)
	}

	resp.Message = string(data)
	resp.ErrorCode = 0
	return c.JSON(http.StatusOK, resp)
}

func xterm(c echo.Context) error {
	rw := c.Response().Writer
	handleconsolews(rw, c.Request())

	return nil
}
