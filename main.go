package main

import (
	"github.com/go-kit/kit/log/level"
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/oklog/run"
	"os"
	"os/signal"
	"syscall"
	"context"
	"prome-route/pkg"
)

func main() {

	var (
		configFile = kingpin.Flag("config.file", "prome-route configuration file path.").Default("prome-route.yml").String()
	)

	// init logger
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("prome-route"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	// new ctxall
	ctxAll, cancelAll := context.WithCancel(context.Background())
	// init config
	sc, err := pkg.LoadFile(*configFile, logger)
	if err != nil {
		level.Error(logger).Log("msg", "Load Config File Failed ....")
		return
	}

	pkg.InitLabelRegStr(sc.ReplaceLabelName)
	pkg.SetDefaultVar(sc)
	pkg.RromeServerMap = make(map[string]string)
	pkg.RromeServerMap = sc.PromeServers
	var g run.Group
	{
		// Termination handler.
		term := make(chan os.Signal, 1)
		signal.Notify(term, os.Interrupt, syscall.SIGTERM)
		cancel := make(chan struct{})
		g.Add(

			func() error {
				select {
				case <-term:
					level.Warn(logger).Log("msg", "Received SIGTERM, exiting gracefully...")
					cancelAll()
					return nil
					//TODO clean work here
				case <-cancel:
					level.Warn(logger).Log("msg", "server finally exit...")
					return nil
				}
			},
			func(err error) {
				close(cancel)

			},
		)
	}

	{
		// metrics web handler.
		g.Add(func() error {
			//<-indexFlushDoneChan
			level.Info(logger).Log("msg", "start web service Listening on address", "address", sc.Http.ListenAddr)
			//gin.SetMode(gin.ReleaseMode)
			//routes := gin.Default()
			//routes := gin.New()
			errchan := make(chan error)

			go func() {
				errchan <- pkg.StartGin(sc.Http.ListenAddr)
			}()
			select {
			case err := <-errchan:
				level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
				return err
			case <-ctxAll.Done():
				level.Info(logger).Log("msg", "Web service Exit..")
				return nil

			}

		}, func(err error) {
			cancelAll()
		})
	}
	g.Run()
}
