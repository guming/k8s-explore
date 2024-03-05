package frp

import (
	"context"
	"fmt"
	"github.com/fatedier/frp/client"
	"k8s-explore/frp/models"
	"k8s-explore/frp/notify"
	"k8s-explore/frp/utils"
)

type ServicePair struct {
	Name    string
	Service models.Service
}

func Execute(common models.Common, servicePair []ServicePair) (cancelChan *notify.CancellationChannel, err error) {

	tomlString := createToml(common, servicePair)

	cfg, pxyCfgs, visitorCfgs, err := utils.ParseClientConfig([]byte(tomlString))
	if err != nil {

		return nil, fmt.Errorf("fail to start frpc. more info: '%s'", err.Error())
	}

	svr, errRet := client.NewService(cfg, pxyCfgs, visitorCfgs, "")
	if errRet != nil {
		err = errRet
		return nil, err
	}

	cancelChan = notify.NewCancellationChannelWithCallback(func() {
		svr.Close()
	})

	go func() {
		err = svr.Run(context.Background())
		cancelChan.CancelWithReason(err) // if finish with reason
	}()

	return cancelChan, nil
}

func createToml(common models.Common, servicePair []ServicePair) string {
	frpConfig := models.FrpcClientConfig{
		"common": common,
	}

	for _, element := range servicePair {

		frpConfig[element.Name] = element.Service
	}

	tomlString, _ := utils.Marshal(frpConfig)

	return tomlString
}
