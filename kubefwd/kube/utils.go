package kube

import (
	"fmt"
	"github.com/ahmetb/go-linq/v3"
	frpc "k8s-explore/frp"
	"k8s-explore/frp/constants"
	frpmodels "k8s-explore/frp/models"
	"k8s-explore/kubetunnel"

	v1 "k8s.io/api/core/v1"
	"strconv"
)

func ToKubeTunnelResourceSpec(ctx *ServiceContext, podLabels map[string]string) kubetunnel.KubeTunnelResourceSpec {

	var ports []string

	linq.From(ctx.Ports).Select(func(kubePort interface{}) interface{} {
		return strconv.Itoa(int(kubePort.(v1.ServicePort).Port))
	}).ToSlice(&ports)

	podLabels[constants.KubetunnelSlug] = ctx.ServiceName

	return kubetunnel.KubeTunnelResourceSpec{
		Ports:       kubetunnel.Ports{Values: ports},
		ServiceName: ctx.ServiceName,
		PodLabels:   podLabels,
	}
}

func ToFRPClientPairs(localIP string, remotePortByLocal map[string]string, ctx *ServiceContext) []frpc.ServicePair {

	var servicePairs []frpc.ServicePair

	localPortByRemote := make(map[string]string)

	for k, v := range remotePortByLocal {
		localPortByRemote[v] = k
	}

	linq.From(ctx.Ports).Select(func(kubePort interface{}) interface{} {

		port := strconv.Itoa(int(kubePort.(v1.ServicePort).Port))

		localPort := localPortByRemote[port]

		if localPort == "" {
			return nil // port not found in map
		}

		return frpc.ServicePair{
			Name: fmt.Sprintf("%s-%s", ctx.ServiceName, port),
			Service: frpmodels.Service{
				Type:       "tcp",
				RemotePort: port,
				LocalIP:    localIP,
				LocalPort:  localPort, //TODO: check what happen if port is not found in map
			},
		}

	}).Where(func(servicePair interface{}) bool {
		return servicePair != nil
	}).ToSlice(&servicePairs)

	return servicePairs
}
