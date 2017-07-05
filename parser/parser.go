package parser

import (
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
)

var updates = make(chan []*Record)
var mux sync.Mutex

var dockerClient *client.Client

//Record of OpenVPN log
type Record struct {
	IP         string
	Container  string
	Name       string
	MacAddress string
	Network    string
}

//GetChannel the channel used to notify updates
func GetChannel() chan []*Record {
	return updates
}

//return a docker client
func getClient() (*client.Client, error) {

	if dockerClient == nil {
		cli, err := client.NewEnvClient()
		if err != nil {
			return nil, err
		}
		dockerClient = cli
	}

	return dockerClient, nil
}

// ListenEvents watches docker events an handle state modifications
func ListenEvents() error {

	cli, err := getClient()
	if err != nil {
		return err
	}

	f := filters.NewArgs()

	msgChan, errChan := cli.Events(context.Background(), types.EventsOptions{
		Filters: f,
	})

	go func() {
		for {
			select {
			case event := <-msgChan:
				if &event != nil {

					log.Infof("Event recieved: %s %s ", event.Action, event.Type)
					if event.Actor.Attributes != nil {

						name := event.Actor.Attributes["name"]
						switch event.Action {
						case "start":
							log.Debugf("Container started %s", name)
							FetchNetworks()
							break
						case "die":
							log.Debugf("Container exited %s", name)
							FetchNetworks()
							break
						}
					}
				}
			case err := <-errChan:
				if err != nil {
					log.Errorf("Error event recieved: %s", err.Error())
				}
			}
		}
	}()

	return nil
}

//FetchNetworks fetch the networks and produce an list of records
func FetchNetworks() error {

	mux.Lock()
	records := make([]*Record, 0)

	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	networks, err := cli.NetworkList(context.Background(), types.NetworkListOptions{
		Filters: filters.NewArgs(),
	})

	if err != nil {
		return err
	}

	for _, network := range networks {
		log.Debugf("Network %s", network.Name)
		for _, host := range network.Containers {
			log.Debugf("Container %s at %s", host.Name, host.IPv4Address)

			record := &Record{
				IP:         strings.Split(host.IPv4Address, "/")[0],
				Name:       host.Name + "." + network.Name,
				MacAddress: host.MacAddress,
				Network:    network.Name,
				Container:  host.Name,
			}

			records = append(records, record)
		}
	}

	log.Debugf("Found %d records", len(records))
	updates <- records

	mux.Unlock()

	return nil
}
