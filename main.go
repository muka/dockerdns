package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/Sirupsen/logrus"

	"github.com/muka/dockerdns/ddns"
	"github.com/muka/dockerdns/parser"

	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()

	app.Name = "dockerdns"
	app.Usage = "Expose docker service to local DNS"

	app.Flags = []cli.Flag{
		// cli.StringFlag{
		// 	Name:   "docker, d",
		// 	Value:  "unix:///var/run/docker.sock",
		// 	Usage:  "Docker UNIX sock",
		// 	EnvVar: "DOCKER_SOCK",
		// },
		cli.StringFlag{
			Name:   "out, o",
			Value:  "",
			Usage:  "Set the output file of a hosts-like formatted list of clients",
			EnvVar: "OUT_FILE",
		},
		cli.StringFlag{
			Name:   "domain, d",
			Value:  "docker.lan",
			Usage:  "Set the default domain to append to each host name",
			EnvVar: "DOMAIN",
		},
		cli.StringFlag{
			Name:   "ddns-host",
			Value:  "127.0.0.1:5551",
			Usage:  "DDNS API host",
			EnvVar: "DDNS_HOST",
		},
		cli.BoolFlag{
			Name:   "ddns",
			Usage:  "Enable ddns sync",
			EnvVar: "DDNS",
		},
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "Enable debugging logs",
			EnvVar: "DEBUG",
		},
	}

	app.Action = func(c *cli.Context) error {

		debug := c.Bool("debug")
		// docker := c.String("docker")
		out := c.String("out")
		domain := c.String("domain")
		ddnsFlag := c.Bool("ddns")
		ddnsHost := c.String("ddns-host")

		if debug {
			log.SetLevel(log.DebugLevel)
		}

		if out == "" && !ddnsFlag {
			log.Info("Please provide at least an option between --out and --ddns")
			return nil
		}
		if ddnsFlag && ddnsHost == "" {
			log.Info("Please provide the --ddns-host option")
			return nil
		}

		if ddnsFlag && ddnsHost != "" {
			ddns.CreateClient(ddnsHost)
		}

		go func() {
			updates := parser.GetChannel()
			for {
				select {
				case records := <-updates:

					if out != "" {
						err := updateHosts(records, domain, out)
						if err != nil {
							log.Errorf("Error saving %s: %s", out, err.Error())
						}
					}

					if ddnsFlag && ddnsHost != "" {
						ddns.Compare(records, domain)
					}
				}
			}
		}()

		parser.ListenEvents()
		parser.FetchNetworks()
		select {}
		// return nil
	}

	app.Run(os.Args)

}

func updateHosts(records []*parser.Record, domain string, out string) error {

	log.Debugf("Updating configuration, %d records", len(records))

	var buffer bytes.Buffer
	for _, record := range records {
		c := fmt.Sprintf("%s %s.%s", record.IP, record.Name, domain)

		log.Debugf("Add line %s", c)
		buffer.WriteString(c)
		buffer.WriteString("\n")
	}

	log.Debugf("Storing to file %s", out)
	return ioutil.WriteFile(out, buffer.Bytes(), 0644)
}
