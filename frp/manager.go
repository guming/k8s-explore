package frp

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/txn2/kubefwd/pkg/fwdport"
	"k8s-explore/frp/constants"
	"k8s-explore/frp/models"
	"k8s-explore/frp/notify"
	"k8s-explore/frp/notify/killsignal"
	"k8s-explore/frp/utils"
	"strings"
	"time"
)

type Manager struct {
	Common      models.Common
	ServicePair []ServicePair
}

func NewManager(common models.Common, servicePair []ServicePair, hostFile *fwdport.HostFileWithLock) *Manager {

	return &Manager{
		common,
		servicePair,
	}
}

func (m *Manager) RunFRPc() {

	for killsignal.HasKillSignaled() == false {

		m.WaitForLocalPortToBecomeAvailable()

		cancelChan, err := Execute(m.Common, m.ServicePair)
		if err != nil {
			log.Panicf("unable to start frp client, err: %s", err.Error())
		}

		go func() {
			m.WaitForLocalPortToBecomeUnavailableAndCancel(cancelChan)
		}()

		cancelChan.WaitForCancellation()
	}

	log.Info("exit frp manager")

}

func (m *Manager) WaitForLocalPortToBecomeAvailable() {

	// TODO: assuming we are using a single service at the time
	host := m.ServicePair[0].Service.LocalIP
	port := m.ServicePair[0].Service.LocalPort

	for killsignal.HasKillSignaled() == false && utils.IsAvailable(host, port) == false {
		time.Sleep(time.Millisecond * 500)
	}
}

func (m *Manager) WaitForLocalPortToBecomeUnavailableAndCancel(channel *notify.CancellationChannel) {

	// TODO: assuming we are using a single service at the time
	host := m.ServicePair[0].Service.LocalIP
	port := m.ServicePair[0].Service.LocalPort

	for killsignal.HasKillSignaled() == false && channel.IsCancelled() == false && utils.IsAvailable(host, port) {
		time.Sleep(time.Millisecond * 500)
	}

	if killsignal.HasKillSignaled() == false && channel.IsCancelled() == false {
		channel.CancelWithReason(fmt.Errorf("service is unavailable at address: %s:%s, shutting down frpc", host, port))
	}
}

func ChangeHostToKubeTunnel(hostFile *fwdport.HostFileWithLock, kubeTunnelServerDns string) {

	originalServiceDns := strings.Replace(kubeTunnelServerDns, fmt.Sprintf("%s-", constants.KubetunnelSlug), "", 1)

	utils.ReplaceAddressForHost(hostFile.Hosts, originalServiceDns, kubeTunnelServerDns)
}
