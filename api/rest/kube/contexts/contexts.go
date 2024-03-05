package contexts

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"k8s-explore/api"
	"k8s-explore/kubeclient"
	"net/http"
)

type Handler struct {
	api.Handler
	clientPool *kubeclient.ClientPool
}

func NewHandler(p *kubeclient.ClientPool, logger *logrus.Entry) *Handler {
	return &Handler{
		Handler:    api.NewHandler("kube/contexts", logger),
		clientPool: p,
	}
}

type Context struct {
	Name       string `json:"name"`
	User       string `json:"user"`
	Cluster    string `json:"cluster"`
	ClusterUID string `json:"clusterUID"`
	Namespace  string `json:"namespace"`
	Current    bool   `json:"current"`
}

func (h *Handler) List(c *gin.Context) {
	cs := []Context{}
	for _, kcxt := range h.clientPool.Contexts() {
		cs = append(cs, Context{
			Name:       kcxt.Name(),
			User:       kcxt.User(),
			Cluster:    kcxt.Cluster(),
			ClusterUID: kcxt.ClusterUID(),
			Namespace:  kcxt.Namespace(),
			Current:    kcxt.Name() == h.clientPool.CurrentContext().Name(),
		})
	}
	c.JSON(http.StatusOK, cs)
}
