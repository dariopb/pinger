package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	pinger "github.com/dariopb/pinger/pkg"
	log "github.com/sirupsen/logrus"

	"github.com/urfave/cli/v2"
)

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

// server
var port int
var loglevelstr string

var enableUpload bool
var token string
var lbapiendpoint string
var insecuretls bool
var servicename string
var lbToken string
var frontendport int

var banner = `
Pinger
`

func main() {
	fmt.Println(banner)

	app := &cli.App{
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "loglevel",
				Aliases:     []string{"l"},
				Value:       "info",
				Usage:       "debug level, one of: info, debug",
				EnvVars:     []string{"LOGLEVEL"},
				Destination: &loglevelstr,
			},
			&cli.IntFlag{
				Name:        "port",
				Aliases:     []string{"p"},
				Value:       8080,
				Usage:       "port for the HTTP rest endpoint",
				EnvVars:     []string{"PORT"},
				Destination: &port,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "apiendpoint",
				Aliases:     []string{"a"},
				Value:       "",
				Usage:       "API endpoint in the form: hostname:port",
				EnvVars:     []string{"LB_API_ENDPOINT"},
				Destination: &lbapiendpoint,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "lbtoken",
				Aliases:     []string{"t"},
				Value:       "",
				Usage:       "shared secret for lb authorization",
				EnvVars:     []string{"LB_TOKEN"},
				Destination: &lbToken,
				Required:    false,
			},
			&cli.IntFlag{
				Name:        "frontendport",
				Aliases:     []string{"f"},
				Value:       0,
				DefaultText: "auto",
				Usage:       "frontend port where the service is going to be exposed (endpoint will be apiendpoint:serviceport)",
				EnvVars:     []string{"LB_PORT"},
				Destination: &frontendport,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "servicename",
				Aliases:     []string{"s"},
				Usage:       "service name string",
				EnvVars:     []string{"LB_SERVICE_NAME"},
				Destination: &servicename,
				Required:    false,
			},
			&cli.BoolFlag{
				Name:        "insecuretls",
				Value:       true,
				Usage:       "allow skip checking server CA/hostname",
				EnvVars:     []string{"LB_INSECURE_TLS"},
				Destination: &insecuretls,
				Required:    false,
			},
			&cli.BoolFlag{
				Name:        "enableUpload",
				Value:       false,
				Usage:       "allow skip checking server CA/hostname",
				EnvVars:     []string{"ENABLE_UPLOAD"},
				Destination: &enableUpload,
				Required:    false,
			},
			&cli.StringFlag{
				Name:        "token",
				Aliases:     []string{"k"},
				Value:       "",
				Usage:       "secret to access http resources",
				EnvVars:     []string{"TOKEN"},
				Destination: &token,
				Required:    false,
			},
		},
		Name:   "pinger",
		Usage:  "Container multi-utility",
		Action: server,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func server(ctx *cli.Context) error {
	printVersion()

	loglevel := log.InfoLevel
	if l, err := log.ParseLevel(loglevelstr); err == nil {
		loglevel = l
	}

	//log.AddHook(ProcessCounter)
	//log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetLevel(loglevel)
	log.SetOutput(os.Stdout)

	err := pinger.NewPinger(port, enableUpload, token, lbapiendpoint, servicename, frontendport, lbToken)
	if err != nil {
		log.Fatalf("failed to start pinger: ", err)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c

	return err
}
