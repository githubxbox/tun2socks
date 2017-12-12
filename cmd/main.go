package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/FlowerWrong/netstack/tcpip"
	"github.com/FlowerWrong/tun2socks/netstack"
	"github.com/FlowerWrong/tun2socks/tun2socks"
	"github.com/FlowerWrong/tun2socks/util"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	// log with file and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Use CPU number", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU())

	config := flag.String("config", "", "config file")
	flag.Parse()
	configFile := *config
	if configFile == "" {
		configFile = flag.Arg(0)
		if configFile == "" {
			if runtime.GOOS == "linux" {
				configFile = "/home/" + os.Getenv("SUDO_USER") + "/.tun2socks/config.ini"
			} else if runtime.GOOS == "darwin" {
				configFile = "/Users/" + os.Getenv("SUDO_USER") + "/.tun2socks/config.ini"
			}
		}
	}
	log.Println("config file is", configFile)

	app := new(tun2socks.App)
	app.Config(configFile).NewTun().AddRoutes().SignalHandler()

	var proto tcpip.NetworkProtocolNumber
	proto = netstack.NewNetstack(app)

	app.WG.Add(1)
	go netstack.NewTCPEndpointAndListenIt(proto, app)
	if app.Cfg.Udp.Enabled {
		app.WG.Add(1)
		_, err := app.Cfg.UdpProxy()
		if err != nil {
			log.Fatal("Get udp socks 5 proxy failed", err)
		}
		go netstack.NewUDPEndpointAndListenIt(proto, app)
	}
	if app.Cfg.Dns.DnsMode == "fake" {
		app.WG.Add(1)
		go func(app *tun2socks.App) {
			util.UpdateDNSServers(true)
			app.FakeDns.Serve()
			app.WG.Done()
		}(app)

		app.WG.Add(1)
		go func(app *tun2socks.App) {
			// clearExpiredNonProxyDomain and clearExpiredDomain
			app.FakeDns.DnsTablePtr.Serve()
			app.WG.Done()
		}(app)
	}

	app.WG.Wait()
}
