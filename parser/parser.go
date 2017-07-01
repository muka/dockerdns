package parser

import (
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
)

var dockerClient *client.Client
var updates = make(chan []*Record, 10)
var mux sync.Mutex

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

// return a docker client instance
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

//ListenEvents Listen for docker Events
func ListenEvents() error {

	cli, err := getClient()
	if err != nil {
		log.Errorf("Failed to get docker client: %s", err.Error())
		return err
	}

	ctx := context.Background()

	f := filters.NewArgs()
	// f.Add("label", "example=1")

	// <-chan events.Message, <-chan error
	msgChan, errChan := cli.Events(ctx, types.EventsOptions{
		Filters: f,
	})

	for {
		select {
		case event := <-msgChan:
			if &event != nil {

				// log.Debugf("Event recieved: %s %s", event.Action, event.Type)

				switch event.Type {
				case events.NetworkEventType:

					switch event.Action {
					case "connect":
					case "disconnect":
						WaitForContainer()
						break
					}
					break
				case events.ContainerEventType:
					// name := event.Actor.Attributes["name"]
					// log.Debugf("Container %s event recieved: %s", name, event.Action)

					switch event.Action {
					// case "create":
					case "start":
					case "restart":
					case "stop":
					// case "kill":
					case "die":
						// case "destroy":
						WaitForContainer()
						break
					}
					break
				}

			}
		case err := <-errChan:
			if err != nil {
				log.Errorf("Error event recieved: %s", err.Error())
			}
		}
	}

}

var ticker *time.Ticker
var eventCounter uint32

//WaitForContainer Plan a network lookup after a while based on event emitted
func WaitForContainer() error {

	log.Debug("Add event")
	eventCounter++

	if ticker == nil {
		log.Debug("Started ticker")
		ticker = time.NewTicker(time.Millisecond * 500)
		go func() {
			for range ticker.C {

				eventCounter--

				log.Debugf("Tick, events len %d", eventCounter)

				if eventCounter <= 0 {
					FetchNetworks()
					eventCounter = 0
					ticker.Stop()
					ticker = nil
					log.Debug("Stopped ticker")
					break
				}
			}
		}()
	}

	return nil
}

//FetchNetworks fetch the networks and produce an list of records
func FetchNetworks() error {

	// mux.Lock()
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
		log.Debugf("Network %s has %d containers", network.Name, len(network.Containers))
		log.Debug(network.Containers)

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

	// mux.Unlock()

	return nil
}
