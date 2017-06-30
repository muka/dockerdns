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
