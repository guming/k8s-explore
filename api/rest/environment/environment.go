package environment

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"k8s-explore/api"
	"k8s-explore/kubeclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"net/http"
)

type Frontend struct {
	Debug bool `json:"debug"`
}

type Parameters struct {
	InstallInfra bool     `json:"installInfra"`
	Frontend     Frontend `json:"frontend"`
}

type CompositionSelector struct {
	MatchLabels map[string]string `json:"matchLabels"`
}

type WriteConnectionSecretToRef struct {
	Name string `json:"name"`
}

type Condition struct {
	Status string `json:"status"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type EnvironmentStatus struct {
	Conditions []Condition `json:"conditions"`
}

type ResourceRef struct {
	Name string `json:"name,omitempty"`
}

type EnvironmentSpec struct {
	WriteConnectionSecretToRef WriteConnectionSecretToRef `json:"writeConnectionSecretToRef,omitempty"`
	Parameters                 Parameters                 `json:"parameters"`
	CompositionSelector        CompositionSelector        `json:"compositionSelector,omitempty"`
	ResourceRef                *ResourceRef               `json:"resourceRef,omitempty"`
}

type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec"`
	Status EnvironmentStatus `json:"status"`
}
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Environment `json:"items"`
}

type Handler struct {
	api.Handler
	clientPool *kubeclient.ClientPool
}

func NewHandler(p *kubeclient.ClientPool, logger *logrus.Entry) *Handler {
	return &Handler{
		Handler:    api.NewHandler("kube/environment", logger),
		clientPool: p,
	}
}

func (h *Handler) kubeClient(c *gin.Context, logger *logrus.Entry) (dynamic.Interface, error) {
	kctx := h.clientPool.CurrentContext()
	client, err := kctx.DynamicClient()
	if err != nil {
		logger.WithError(err).Error("can not get k8s client for the context")
		c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	return client, nil
}

func (h *Handler) List(c *gin.Context) {
	logger := h.Logger(c).WithField("method", "List")
	client, err := h.kubeClient(c, logger)
	if err != nil {
		return
	}
	list, err := client.Resource(schema.GroupVersionResource{
		Group:    "salaboy.com",
		Version:  "v1alpha1",
		Resource: "environments",
	}).Namespace("default").List(c.Request.Context(), metav1.ListOptions{
		FieldSelector: c.Query("fieldSelector"),
		LabelSelector: c.Query("labelSelector"),
	})
	if err != nil {
		logger.
			WithError(err).
			Error("Couldn't list Kubernetes objects")
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"error": "internal server error"},
		)
		return
	}
	result, err := convertUnstructuredListToEnvironmentList(list)
	c.JSON(http.StatusOK, result)
}

func convertUnstructuredListToEnvironmentList(unstructuredList *unstructured.UnstructuredList) ([]Environment, error) {
	var environmentList []Environment

	for _, item := range unstructuredList.Items {
		// 将 Unstructured 对象反序列化为 Environment 对象
		var env Environment
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &env)
		if err != nil {
			return nil, err
		}

		// 将 Environment 对象添加到列表
		environmentList = append(environmentList, env)
	}

	return environmentList, nil
}
