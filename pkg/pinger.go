package pinger

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	tunnel "github.com/dariopb/goreverselb/pkg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
)

type PingerCtx struct {
	e             *echo.Echo
	cmd           *XtermObj
	uploadDirName string
}
type CommandObj struct {
	Target string `json:"target,omitempty" query:"target"`
	Text   string `json:"text,omitempty" query:"text"`
}

type XtermObj struct {
	Cmd         string `json:"cmd,omitempty" query:"cmd"`
	SessionName string `json:"sessionName,omitempty" query:"sessionName"`

	allowed bool
}

type ResponseObj struct {
	ErrorCode     int    `json:"errorCode"`
	HTTPErrorCode int    `json:"httpErrorCode,omitempty"`
	Message       string `json:"message,omitempty"`
}

func (c *XtermObj) Evaluate() {

	if c.Cmd == "" {
		os := runtime.GOOS
		switch os {
		case "windows":
			c.Cmd = "cmd.exe"
		case "darwin":
		case "linux":
			c.Cmd = "/bin/bash"
		default:
			c.Cmd = "/bin/bash"
		}
	}
}

func NewPinger(port int, enableUpload bool, enableXterm bool, token string, lbEndpointAPI string, lbServicename string, lbFrontendPort int, lbToken string) error {
	fmt.Printf("starting pinger on [%d]...\n", port)

	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			log.Debugf("IP: %s", addr)
		}
	}

	if lbEndpointAPI != "" && lbServicename != "" {
		td := tunnel.TunnelData{
			ServiceName:          lbServicename,
			BackendAcceptBacklog: 1,
			FrontendData: tunnel.FrontendData{
				Port: lbFrontendPort,
			},
			TargetPort:      port,
			TargetAddresses: []string{"127.0.0.1"},
			Token:           lbToken,
		}

		_, err = tunnel.NewMuxTunnelClient(lbEndpointAPI, td)
		if err != nil {
			panic(err)
		}
	} else {
		log.Info("Not starting lb association")
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = false

	pingerCtx := &PingerCtx{
		e:             e,
		uploadDirName: "files",
	}
	pingerCtx.cmd = &XtermObj{}
	pingerCtx.cmd.Evaluate()

	if enableUpload {
		log.Infof("Enabling upload support on: /%s", pingerCtx.uploadDirName)
		os.Mkdir(pingerCtx.uploadDirName, 0666)

		filesGroup := e.Group("/" + pingerCtx.uploadDirName)
		filesGroup.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:       "files", //pingerCtx.uploadDirName,
			Browse:     true,
			IgnoreBase: true,
		}))

		e.Static("/upload", "web/upload.html")
		e.POST("/uploadMultipart", pingerCtx.uploadMultipart)
	} else {
		log.Info("Uploads are disabled")
	}

	if token != "" {
		e.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
			KeyLookup: "query:token",
			Validator: func(key string, c echo.Context) (bool, error) {
				ok := key == token

				if !ok {
					log.Warnf("Non-authenticated request coming for [%s] from: %s", c.Request().RequestURI, c.Request().RemoteAddr)
				}
				return ok, nil
			},
			Skipper: func(c echo.Context) bool {
				p := c.Path()
				return p == "/" || p == "/web*" || p == "/xterm/ws" || p == "/favicon.ico" || p == "/uploadMultipart" || p == "/files/*"
			},
		}))
	} else {
		log.Warn("Token not provided, requests will be allowed for anybody")
	}

	e.Static("/web", "web")

	// Routes
	e.GET("/", hello)
	e.GET("/hello", hello)
	e.GET("/id", pingerCtx.id)
	e.GET("/proxy", proxy)
	e.GET("/resolveName", resolveDns)

	if enableXterm {
		e.File("/xterm", "web/index.html", pingerCtx.m)
		e.GET("/xterm/ws", pingerCtx.xtermws)
	}

	// Start server
	go func() {
		err := e.Start(fmt.Sprintf(":%d", port))
		if err == nil {
			log.Fatalf("Failed starting http server: %v", err)
		}
	}()

	return nil
}

func (p *PingerCtx) m(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cmd := new(XtermObj)
		err := c.Bind(cmd)
		if err != nil {
			resp := ResponseObj{
				ErrorCode: -1,
			}
			resp.Message = fmt.Sprintf("Error binding: [%s]: Expecting: cmd=[cmd to run]", err)
			return c.JSON(http.StatusOK, resp)
		}

		cmd.Evaluate()
		cmd.allowed = true
		p.cmd = cmd

		log.WithField("cmd", cmd.Cmd).Info("Setting cmd")

		// Disable cache for this one
		epoch := time.Unix(0, 0).Format(time.RFC1123)

		noCacheHeaders := map[string]string{
			"Expires":         epoch,
			"Cache-Control":   "no-cache, private, max-age=0",
			"Pragma":          "no-cache",
			"X-Accel-Expires": "0",
		}

		res := c.Response()
		for k, v := range noCacheHeaders {
			res.Header().Set(k, v)
		}

		return next(c)
	}
}

func hello(c echo.Context) error {
	hostname, _ := os.Hostname()
	banner := fmt.Sprintf("Pinger: I'm alive on [%s] (%s on %s/%s). Supports: dns, proxy, vars, LB (LB_API_ENDPOINT/LB_TOKEN/LB_PORT/LB_SERVICE_NAME), xterm (linux/windows, ENABLE_XTERM), upload via POST (ENABLE_UPLOAD), dapr invoke.",
		hostname, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	banner += "<h3>Query strings</h3><snap>"
	banner += "<li>token: access token (if server neededs it))</li>"
	banner += "<li>cmd: command to run for xterm</li>"
	banner += "<li>fontFamily [Cascadia Code/Cascadia Mono/Roboto Mono/Ubuntu Mono]/fontSize/fontWeight for xterm console</li>"
	banner += "<li>Target: the target for DNS resolution (dns name) or proxy (http url to proxy to)</li>"
	banner += "<li></li>"
	banner += "</snap>"
	banner += "<h3>Available routes</h3><snap>"
	routes := c.Echo().Routes()
	for _, route := range routes {
		banner = banner + route.Path + ", "
	}
	banner += "</snap>"

	banner += "<h3>IP configuration</h3><snap style='display: block;background-color:lightblue;'>"

	links, err := net.Interfaces()
	if err == nil {

		for _, li := range links {
			banner += "<li>" + li.Name + ": " + li.HardwareAddr.String() + "<br>"
			addresses, err := li.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addresses {
				banner += "&nbsp;&nbsp;&nbsp;&nbsp;" + addr.String() + "<br>"
			}
			banner += "</li>"
		}
	}
	banner += "</snap>"

	banner += "<h3>Variables</h3><span style='display: block;background-color:yellow;'><li>"
	banner = fmt.Sprintf("%s  %s", banner, strings.Join(os.Environ(), "</li><li>"))
	banner += "</li></snap>"

	return c.HTML(http.StatusOK, banner)
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

func (p *PingerCtx) xtermws(c echo.Context) error {
	if !p.cmd.allowed {
		return fmt.Errorf("Not authorized")
	}
	rw := c.Response().Writer
	handleconsolews(p.cmd.Cmd, rw, c.Request())

	return nil
}

func (p *PingerCtx) id(c echo.Context) error {
	colorArray := []string{
		"#7B68EE", "#EE82EE", "#FFA07A", "#FFD700",
		"#7FFF00", "#9ACD32", "#20B2AA", "#7FFFD4",
		"#87CEFA", "#1E90FF", "#808000", "#D3D3D3",
	}

	msg := ""
	sum := 0

	msg += "<snap style='display: block;font-size: 20;font-weight: bold'>"
	links, err := net.Interfaces()
	if err == nil {
		for _, li := range links {
			msg += "<li>" + li.Name + ": " + li.HardwareAddr.String() + "<br>"
			addresses, err := li.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addresses {
				msg += "&nbsp;&nbsp;&nbsp;&nbsp;" + addr.String() + "<br>"
			}
			msg += "</li>"
		}
	}
	msg += "</snap>"

	color := colorArray[sum%len(colorArray)]
	html := fmt.Sprintf("<html><body style='background-color:%s;'>%s</body></html>", color, msg)

	return c.HTML(http.StatusOK, html)
}

func (p *PingerCtx) uploadMultipart(c echo.Context) error {
	// Multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	files := form.File["files"]

	for _, file := range files {
		// Source
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		// Destination
		dst, err := os.Create(path.Join(p.uploadDirName, file.Filename))
		if err != nil {
			return err
		}
		defer dst.Close()

		// Copy
		if _, err = io.Copy(dst, src); err != nil {
			return err
		}

	}

	return c.HTML(http.StatusOK, fmt.Sprintf("<p>Uploaded successfully %d files.</p>", len(files)))
}
