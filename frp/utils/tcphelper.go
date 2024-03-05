package utils

import (
	log "github.com/sirupsen/logrus"
	"github.com/txn2/txeh"
	"k8s-explore/frp/constants"
	"net"
	"strings"
	"time"
)

func IsAvailable(host string, port string) bool {

	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		log.Debugf("connection error: %s", err)
		return false
	}

	defer conn.Close()
	log.Debugf("connection succeeded: %s", net.JoinHostPort(host, port))

	return true
}

func ReplaceAddressForHost(hosts *txeh.Hosts, srcHost, dstHost string) {

	log.Infof("replacing host: '%s' with: '%s'", srcHost, dstHost)

	found, newAddr, _ := hosts.HostAddressLookup(dstHost, txeh.IPFamilyV4)

	if found == false {
		log.Panicf("unable to locate host: %s in hosts file. please run %s again.", dstHost, constants.KubetunnelSlug)
	}

	for _, line := range *hosts.GetHostFileLines() {

		for _, hostname := range line.Hostnames {

			if strings.EqualFold(srcHost, hostname) {

				hosts.RemoveAddress(line.Address)
				hosts.AddHosts(newAddr, line.Hostnames)

				err := hosts.Save()
				if err != nil {
					log.Panicf("unable to save host: %s in hosts file. please run %s again. internal err: %s", dstHost, constants.KubetunnelSlug, err.Error())
				}

				break
			}
		}
	}
}

func HostsCleanup(hosts *txeh.Hosts) {

	log.Info("cleaning up all entries containing .kubetunnel host")

	for _, line := range *hosts.GetHostFileLines() {

		for _, hostname := range line.Hostnames {

			if strings.Contains(hostname, constants.KubetunnelSlug) {

				hosts.RemoveAddress(line.Address)
				break
			}
		}
	}
}
