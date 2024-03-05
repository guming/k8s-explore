package frp_test

import (
	"github.com/stretchr/testify/assert"
	"k8s-explore/frp"
	"k8s-explore/frp/models"
	"testing"
)

func TestInstallingKubetunnelGC(t *testing.T) {

	common := models.Common{
		ServerAddress: "localhost",
		ServerPort:    "7001",
	}

	svc := []frp.ServicePair{
		{
			Name: "google",
			Service: models.Service{
				Type:       "tcp",
				RemotePort: "80",
				LocalIP:    "google.com",
				LocalPort:  "80",
			},
		},
		{
			Name: "microsoft",
			Service: models.Service{
				Type:       "tcp",
				RemotePort: "8081",
				LocalIP:    "microsoft.com",
				LocalPort:  "80",
			},
		},
	}

	_, err := frp.Execute(common, svc)

	assert.NoError(t, err)
}
