package frp_test

import (
	"k8s-explore/frp"
	"k8s-explore/frp/models"
	"testing"
)

func TestFrpClientManager(t *testing.T) {

	common := models.Common{
		ServerAddress: "kubetunnel-demo",
		ServerPort:    "7000",
	}

	svc := []frp.ServicePair{
		{
			Name: "local_app",
			Service: models.Service{
				Type:       "tcp",
				RemotePort: "80",
				LocalIP:    "localhost",
				LocalPort:  "22285",
			},
		},
	}

	manager := frp.NewManager(common, svc, nil)

	manager.RunFRPc()

}
