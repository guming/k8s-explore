package objects

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"k8s-explore/api"
	"k8s-explore/kubeclient"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"net/http"
)

type Handler struct {
	api.Handler
	clientPool *kubeclient.ClientPool
}

func NewHandler(clientPool *kubeclient.ClientPool, logger *logrus.Entry) *Handler {
	return &Handler{
		Handler:    api.NewHandler("kube/objects", logger),
		clientPool: clientPool,
	}
}

func (h *Handler) kubeClient(c *gin.Context, logger *logrus.Entry) (dynamic.Interface, error) {
	kctx, err := h.clientPool.Context(c.Param("ctx"))
	if err != nil {
		logger.WithError(err).Error("unknown context")
		c.AbortWithStatusJSON(http.StatusNotFound, map[string]string{"error": "unknown context"})
	}
	client, err := kctx.DynamicClient()
	if err != nil {
		logger.WithError(err).Error("can not get k8s client for the context")
		c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	return client, nil
}

func (h *Handler) Get(c *gin.Context) {
	logger := getLogger(c, h, "Get")
	group := c.Param("group")
	if group == "core" {
		group = ""
	}
	client, err := h.kubeClient(c, logger)
	if err != nil {
		return
	}
	obj, err := client.Resource(schema.GroupVersionResource{
		Group:    group,
		Version:  c.Param("version"),
		Resource: c.Param("resource"),
	}).Namespace(c.Param("namespace")).Get(c.Request.Context(), c.Param("name"), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			c.AbortWithStatusJSON(
				http.StatusNotFound,
				map[string]string{"error": "not found"},
			)
			return
		}

		logger.
			WithError(err).
			Error("Couldn't get Kubernetes object")
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"error": "internal server error"},
		)
		return
	}
	c.JSON(http.StatusOK, obj)
}

func (h *Handler) List(c *gin.Context) {
	logger := getLogger(c, h, "List")

	group := c.Param("group")
	if group == "core" {
		group = ""
	}

	client, err := h.kubeClient(c, logger)
	if err != nil {
		return
	}

	list, err := client.
		Resource(schema.GroupVersionResource{
			Group:    group,
			Version:  c.Param("version"),
			Resource: c.Param("resource"),
		}).
		Namespace(c.Param("namespace")).
		List(c.Request.Context(), metav1.ListOptions{
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

	c.JSON(http.StatusOK, list.Items)
}

func (h *Handler) Update(c *gin.Context) {
	logger := getLogger(c, h, "Update")
	group := c.Param("group")
	if group == "core" {
		group = ""
	}

	obj, err := h.unstructuredObjectFromRequest(c, logger)
	if err != nil {
		return
	}

	client, err := h.kubeClient(c, logger)
	if err != nil {
		return
	}

	obj, err = client.
		Resource(schema.GroupVersionResource{
			Group:    group,
			Version:  c.Param("version"),
			Resource: c.Param("resource"),
		}).
		Namespace(c.Param("namespace")).
		Update(c.Request.Context(), obj, metav1.UpdateOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		logger.
			WithError(err).
			Error("Couldn't update Kubernetes object")
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"error": "internal server error"},
		)
		return
	}

	c.JSON(http.StatusOK, obj)
}

func (h *Handler) Delete(c *gin.Context) {
	logger := getLogger(c, h, "Delete")
	group := c.Param("group")
	if group == "core" {
		group = ""
	}

	client, err := h.kubeClient(c, logger)
	if err != nil {
		return
	}

	err = client.
		Resource(schema.GroupVersionResource{
			Group:    group,
			Version:  c.Param("version"),
			Resource: c.Param("resource"),
		}).
		Namespace(c.Param("namespace")).
		Delete(c.Request.Context(), c.Param("name"), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		logger.
			WithError(err).
			Error("Couldn't delete Kubernetes object")
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"error": "internal server error"},
		)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) unstructuredObjectFromRequest(c *gin.Context, logger *logrus.Entry) (*unstructured.Unstructured, error) {
	body, err := c.GetRawData()
	if err != nil {
		logger.
			WithError(err).
			Error("Couldn't read request body")
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"error": "internal server error"},
		)
		return nil, err
	}
	logger = logger.WithField("body", body)
	jsonBody, err := yaml.ToJSON(body)
	if err != nil {
		logger.
			WithError(err).
			Error("Couldn't convert YAML to JSON")
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"error": "internal server error"},
		)
		return nil, err
	}
	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, jsonBody)
	if err != nil {
		logger.
			WithError(err).
			Error("Couldn't decode Kubernetes object")
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"error": "internal server error"},
		)
		return nil, err
	}
	unstructedObject, ok := obj.(*unstructured.Unstructured)
	if !ok {
		logger.
			WithField("object", obj).
			Error("Couldn't type cast runtime.Object to unstructured.Unstructured")
		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			map[string]string{"error": "internal server error"},
		)
		return nil, err
	}
	return unstructedObject, nil
}

func getLogger(c *gin.Context, h *Handler, methodName string) *logrus.Entry {
	logger := h.Logger(c).
		WithField("method", methodName).
		WithField("context", c.Param("ctx")).
		WithField("group", c.Param("group")).
		WithField("version", c.Param("version")).
		WithField("resource", c.Param("resource")).
		WithField("namespace", c.Param("namespace"))
	return logger
}
