package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

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
	fmt.Println("starting pinger...")

	e := echo.New()
	e.HideBanner = true

	// Routes
	e.GET("/", hello)
	e.GET("/proxy", proxy)
	e.GET("/resolveName", resolveDns)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
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
