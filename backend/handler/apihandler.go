// Copyright 2017 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/networkpolicy"
	"github.com/ycyxuehan/dashboard-gin/backend/utils/httphelper"

	"github.com/ycyxuehan/dashboard-gin/backend/handler/parser"
	// "github.com/ycyxuehan/dashboard-gin/backend/resource/customresourcedefinition/types"

	"github.com/ycyxuehan/dashboard-gin/backend/plugin"

	"golang.org/x/net/xsrftoken"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/ycyxuehan/dashboard-gin/backend/api"
	"github.com/ycyxuehan/dashboard-gin/backend/auth"
	authApi "github.com/ycyxuehan/dashboard-gin/backend/auth/api"
	clientapi "github.com/ycyxuehan/dashboard-gin/backend/client/api"
	"github.com/ycyxuehan/dashboard-gin/backend/errors"
	"github.com/ycyxuehan/dashboard-gin/backend/integration"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/clusterrole"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/clusterrolebinding"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/common"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/configmap"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/container"
	// "github.com/ycyxuehan/dashboard-gin/backend/resource/controller"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/cronjob"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/customresourcedefinition"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/daemonset"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/dataselect"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/deployment"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/event"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/horizontalpodautoscaler"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/ingress"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/job"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/logs"
	ns "github.com/ycyxuehan/dashboard-gin/backend/resource/namespace"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/node"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/persistentvolume"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/persistentvolumeclaim"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/pod"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/replicaset"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/replicationcontroller"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/role"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/rolebinding"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/secret"
	resourceService "github.com/ycyxuehan/dashboard-gin/backend/resource/service"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/serviceaccount"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/statefulset"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/storageclass"
	"github.com/ycyxuehan/dashboard-gin/backend/scaling"
	"github.com/ycyxuehan/dashboard-gin/backend/settings"
	settingsApi "github.com/ycyxuehan/dashboard-gin/backend/settings/api"
	"github.com/ycyxuehan/dashboard-gin/backend/systembanner"
	"github.com/ycyxuehan/dashboard-gin/backend/validation"
)

const (
	// RequestLogString is a template for c log message.
	RequestLogString = "[%s] Incoming %s %s %s c from %s: %s"

	// ResponseLogString is a template for c log message.
	ResponseLogString = "[%s] Outcoming c to %s with %d status code"
)

// APIHandler is a representation of API handler. Structure contains clientapi, Heapster clientapi and clientapi configuration.
type APIHandler struct {
	iManager integration.IntegrationManager
	cManager clientapi.ClientManager
	sManager settingsApi.SettingsManager
}

// TerminalResponse is sent by handleExecShell. The Id is a random session id that binds the original REST c and the SockJS connection.
// Any clientapi in possession of this Id can hijack the terminal session.
type TerminalResponse struct {
	ID string `json:"id"`
}

// CreateHTTPAPIHandler creates a new HTTP handler that handles all cs to the API of the backend.
func CreateHTTPAPIHandler(iManager integration.IntegrationManager, cManager clientapi.ClientManager,
	authManager authApi.AuthManager, sManager settingsApi.SettingsManager,
	sbManager systembanner.SystemBannerManager, e *gin.Engine)error{
	apiHandler := APIHandler{iManager: iManager, cManager: cManager, sManager: sManager}
	// wsContainer := restful.NewContainer()
	// wsContainer.EnableContentEncoding(true)

	// apiV1Ws := new(restful.WebService)

	InstallFilters(e, cManager)

	r := e.Group("/api/v1")

	integrationHandler := integration.NewIntegrationHandler(iManager)
	integrationHandler.Install(r)

	pluginHandler := plugin.NewPluginHandler(cManager)
	pluginHandler.Install(r)

	authHandler := auth.NewAuthHandler(authManager)
	authHandler.Install(r)

	settingsHandler := settings.NewSettingsHandler(sManager, cManager)
	settingsHandler.Install(r)

	systemBannerHandler := systembanner.NewSystemBannerHandler(sbManager)
	systemBannerHandler.Install(r)

	csrftokenGroup := r.Group("/csrftoken")
	csrftokenGroup.GET("/:action", apiHandler.handleGetCsrfToken)
	
	appdeploymentGroup := r.Group("/appdeployment")
	appdeploymentGroup.POST("/", apiHandler.handleDeploy)
	appdeploymentGroup.GET("/protoclos", apiHandler.handleGetAvailableProtocols)
	appdeploymentValidateGroup := appdeploymentGroup.Group("/validate")
	appdeploymentValidateGroup.POST("/name", apiHandler.handleNameValidity)
	appdeploymentValidateGroup.POST("/imagereference", apiHandler.handleImageReferenceValidity)
	appdeploymentValidateGroup.POST("/protocol", apiHandler.handleProtocolValidity)
	r.GET("/appdeploymentfromfile", apiHandler.handleDeployFromFile)
	
	replicationcontrollerGroup := r.Group("/replicationcontroller")
	replicationcontrollerGroup.GET("/", apiHandler.handleGetReplicationControllerList)
	replicationcontrollerGroup.GET("/:namespace", apiHandler.handleGetReplicationControllerList)
	replicationcontrollerGroup.GET("/:namespace/:replicationController", apiHandler.handleGetReplicationControllerDetail)
	replicationcontrollerGroup.POST("/:namespace/:replicationController/update/pod", apiHandler.handleUpdateReplicasCount)
	replicationcontrollerGroup.GET("/:namespace/:replicationController/pod", apiHandler.handleGetReplicationControllerPods)
	replicationcontrollerGroup.GET("/:namespace/:replicationController/event", apiHandler.handleGetReplicationControllerEvents)
	replicationcontrollerGroup.GET("/:namespace/:replicationController/service", apiHandler.handleGetReplicationControllerServices)
	
	replicasetGroup := r.Group("/replicaset")
	replicasetGroup.GET("/", apiHandler.handleGetReplicaSets)
	replicasetGroup.GET("/:namespace", apiHandler.handleGetReplicaSets)
	replicasetGroup.GET("/:namespace/:replicaSet", apiHandler.handleGetReplicaSetDetail)
	replicasetGroup.GET("/:namespace/:replicaSet/pod", apiHandler.handleGetReplicaSetPods)
	replicasetGroup.GET("/:namespace/:replicaSet/event", apiHandler.handleGetReplicaSetEvents)
	replicasetGroup.GET("/:namespace/:replicaSet/service", apiHandler.handleGetReplicaSetServices)

	podGroup := r.Group("/pod")
	podGroup.GET("/", apiHandler.handleGetPods)
	podGroup.GET("/:namespace", apiHandler.handleGetPods)
	podGroup.GET("/:namespace/:pod", apiHandler.handleGetPodDetail)
	podGroup.GET("/:namespace/:pod/container", apiHandler.handleGetPodContainers)
	podGroup.GET("/:namespace/:pod/event", apiHandler.handleGetPodEvents)
	podGroup.GET("/:namespace/:pod/shell/:container", apiHandler.handleExecShell)
	podGroup.GET("/:namespace/:pod/persistentvolumeclaim", apiHandler.handleGetPodPersistentVolumeClaims)

	deploymentGroup := r.Group("/deployment")
	deploymentGroup.GET("/", apiHandler.handleGetDeployments)
	deploymentGroup.GET("/:namespace", apiHandler.handleGetDeployments)
	deploymentGroup.GET("/:namespace/:deployment", apiHandler.handleGetDeploymentDetail)
	deploymentGroup.GET("/:namespace/:deployment/oldreplicaset", apiHandler.handleGetDeploymentOldReplicaSets)
	deploymentGroup.GET("/:namespace/:deployment/event", apiHandler.handleGetDeploymentEvents)
	deploymentGroup.GET("/:namespace/:deployment/newreplicaset", apiHandler.handleGetDeploymentNewReplicaSet)

	scaleGroup := r.Group("/scale")
	scaleGroup.GET("/:kind/:name", apiHandler.handleGetReplicaCount)
	scaleGroup.PUT("/:kind/:name", apiHandler.handleScaleResource)
	scaleGroup.PUT("/:kind/:name/:namespace", apiHandler.handleScaleResource)
	scaleGroup.GET("/:kind/:name/:namespace", apiHandler.handleGetReplicaCount)

	daemonsetGroup := r.Group("/daemonset")
	daemonsetGroup.GET("/", apiHandler.handleGetDaemonSetList)
	daemonsetGroup.GET("/:namespace", apiHandler.handleGetDaemonSetList)
	daemonsetGroup.GET("/:namespace/:daemonset", apiHandler.handleGetDaemonSetDetail)
	daemonsetGroup.GET("/:namespace/:daemonset/pod", apiHandler.handleGetDaemonSetPods)
	daemonsetGroup.GET("/:namespace/:daemonset/service", apiHandler.handleGetDaemonSetServices)
	daemonsetGroup.GET("/:namespace/:daemonset/event", apiHandler.handleGetDaemonSetEvents)

	// r.GET("/:kind/:namespace/:name/horizontalpodautoscaler", apiHandler.handleGetHorizontalPodAutoscalerListForResource)
	horizontalpodautoscalerGroup := r.Group("/horizontalpodautoscaler")
	horizontalpodautoscalerGroup.GET("/", apiHandler.handleGetHorizontalPodAutoscalerList)
	horizontalpodautoscalerGroup.GET("/:namespace", apiHandler.handleGetHorizontalPodAutoscalerList)
	horizontalpodautoscalerGroup.GET("/:namespace/:horizontalpodautoscaler", apiHandler.handleGetHorizontalPodAutoscalerDetail)
	horizontalpodautoscalerGroup.GET("/:namespace/:horizontalpodautoscaler/:kind", apiHandler.handleGetHorizontalPodAutoscalerListForResource)
	jobGroup := r.Group("/job")
	jobGroup.GET("/", apiHandler.handleGetJobList)
	jobGroup.GET("/:namespace", apiHandler.handleGetJobList)
	jobGroup.GET("/:namespace/:name", apiHandler.handleGetJobDetail)
	jobGroup.GET("/:namespace/:name/pod", apiHandler.handleGetJobPods)
	jobGroup.GET("/:namespace/:name/event", apiHandler.handleGetJobEvents)

	cronjobGroup := r.Group("/cronjob")
	cronjobGroup.GET("/", apiHandler.handleGetCronJobList)
	cronjobGroup.GET("/:namespace", apiHandler.handleGetCronJobList)
	cronjobGroup.GET("/:namespace/:name", apiHandler.handleGetCronJobDetail)
	cronjobGroup.GET("/:namespace/:name/job", apiHandler.handleGetCronJobJobs)
	cronjobGroup.GET("/:namespace/:name/event", apiHandler.handleGetCronJobEvents)
	cronjobGroup.GET("/:namespace/:name/trigger", apiHandler.handleTriggerCronJob)
	
	namespaceGroup := r.Group("/namespace")
	namespaceGroup.POST("/", apiHandler.handleCreateNamespace)
	namespaceGroup.GET("/", apiHandler.handleGetNamespaces)
	namespaceGroup.GET("/:name", apiHandler.handleGetNamespaceDetail)
	namespaceGroup.GET("/:name/event", apiHandler.handleGetNamespaceEvents)

	secretGroup := r.Group("/secret")
	secretGroup.POST("/", apiHandler.handleCreateImagePullSecret)
	secretGroup.GET("/", apiHandler.handleGetSecretList)
	secretGroup.GET("/:namespace", apiHandler.handleGetSecretList)
	secretGroup.GET("/:namespace/:name", apiHandler.handleGetSecretDetail)

	configmapGroup := r.Group("/configmap")
	configmapGroup.GET("/", apiHandler.handleGetConfigMapList)
	configmapGroup.GET("/:namespace", apiHandler.handleGetConfigMapList)
	configmapGroup.GET("/:namespace/:configmap", apiHandler.handleGetConfigMapDetail)

	serviceGroup := r.Group("/service")
	serviceGroup.GET("/", apiHandler.handleGetServiceList)
	serviceGroup.GET("/:namespace", apiHandler.handleGetServiceList)
	serviceGroup.GET("/:namespace/:service", apiHandler.handleGetServiceDetail)
	serviceGroup.GET("/:namespace/:service/pod", apiHandler.handleGetServicePods)
	serviceGroup.GET("/:namespace/:service/event", apiHandler.handleGetServiceEvent)

	serviceAccountGroup := r.Group("/serviceaccount")
	serviceAccountGroup.GET("/", apiHandler.handleGetServiceList)
	serviceAccountGroup.GET("/:namespace", apiHandler.handleGetServiceAccountList)
	serviceAccountGroup.GET("/:namespace/:serviceaccount", apiHandler.handleGetServiceAccountDetail)
	serviceAccountGroup.GET("/:namespace/:serviceaccount/secret", apiHandler.handleGetServiceAccountSecrets)
	serviceAccountGroup.GET("/:namespace/:serviceaccount/imagepullsecret", apiHandler.handleGetServiceAccountImagePullSecrets)

	ingressGroup := r.Group("/ingress")
	ingressGroup.GET("/", apiHandler.handleGetIngressList)
	ingressGroup.GET("/:namespace", apiHandler.handleGetIngressList)
	ingressGroup.GET("/:namespace/:name", apiHandler.handleGetIngressDetail)

	networkpolicyGroup := r.Group("networkpolicy")
	networkpolicyGroup.GET("/", apiHandler.handleGetNetworkPolicyList)
	networkpolicyGroup.GET("/:namespace", apiHandler.handleGetNetworkPolicyList)
	networkpolicyGroup.GET("/:namespace/:networkpolicy", apiHandler.handleGetNetworkPolicyDetail)

	statefulsetGroup := r.Group("/statefulset")
	statefulsetGroup.GET("/", apiHandler.handleGetStatefulSetList)
	statefulsetGroup.GET("/:namespace", apiHandler.handleGetStatefulSetList)
	statefulsetGroup.GET("/:namespace/:statefulset", apiHandler.handleGetStatefulSetDetail)
	statefulsetGroup.GET("/:namespace/:statefulset/pod", apiHandler.handleGetStatefulSetPods)
	statefulsetGroup.GET("/:namespace/:statefulset/event", apiHandler.handleGetStatefulSetEvents)

	nodeGroup := r.Group("/node")
	nodeGroup.GET("/", apiHandler.handleGetNodeList)
	nodeGroup.GET("/:namespace", apiHandler.handleGetNodeList)
	nodeGroup.GET("/:namespace/:name", apiHandler.handleGetNodeDetail)
	nodeGroup.GET("/:namespace/:name/pod", apiHandler.handleGetNodePods)
	nodeGroup.GET("/:namespace/:name/event", apiHandler.handleGetNodeEvents)

	rawGroup := r.Group("/_raw/:kind")
	rawGroup.DELETE("/namespace/:namespace/name/:name", apiHandler.handleDeleteResource)
	rawGroup.GET("/namespace/:namespace/name/:name", apiHandler.handleGetResource)
	rawGroup.PUT("/namespace/:namespace/name/:name", apiHandler.handlePutResource)

	rawGroup.DELETE("/name/:name", apiHandler.handleDeleteResource)
	rawGroup.GET("/name/:name", apiHandler.handleGetResource)
	rawGroup.PUT("/name/:name", apiHandler.handlePutResource)

	clusterroleGroup := r.Group("/clusterrole")
	clusterroleGroup.GET("/", apiHandler.handleGetClusterRoleList)
	clusterroleGroup.GET("/:name", apiHandler.handleGetClusterRoleDetail)

	clusterrolebindingGroup := r.Group("/clusterrolebinding")
	clusterrolebindingGroup.GET("/", apiHandler.handleGetClusterRoleBindingList)
	clusterrolebindingGroup.GET("/:name", apiHandler.handleGetClusterRoleBindingDetail)

	roleGroup := r.Group("/role/:namespace")
	roleGroup.GET("/", apiHandler.handleGetRoleList)
	roleGroup.GET("/:name", apiHandler.handleGetRoleDetail)

	rolebindingGroup := r.Group("/rolebinding/:namespace")
	rolebindingGroup.GET("/", apiHandler.handleGetRoleBindingList)
	rolebindingGroup.GET("/:name", apiHandler.handleGetRoleBindingDetail)

	persistentvolumeGroup := r.Group("/persistentvolume")
	persistentvolumeGroup.GET("/", apiHandler.handleGetPersistentVolumeList)
	persistentvolumeGroup.GET("/:persistentvolume", apiHandler.handleGetPersistentVolumeDetail)
	// persistentvolumeGroup.GET("/namespace/:namespace/name/:persistentvolume", apiHandler.handleGetPersistentVolumeDetail)

	persistentvolumeclaimGroup := r.Group("/persistentvolumeclaim")
	persistentvolumeclaimGroup.GET("/", apiHandler.handleGetPersistentVolumeClaimList)
	persistentvolumeclaimGroup.GET("/:namespace", apiHandler.handleGetPersistentVolumeClaimList)
	persistentvolumeclaimGroup.GET("/:namespace/:name", apiHandler.handleGetPersistentVolumeClaimDetail)

	crdGroup := r.Group("/crd")
	crdGroup.GET("/", apiHandler.handleGetCustomResourceDefinitionList)
	crdGroup.GET("/:crd", apiHandler.handleGetCustomResourceDefinitionDetail)
	crdSubGroup := crdGroup.Group("/:crd/:namespace/object")
	crdSubGroup.GET("/",apiHandler.handleGetCustomResourceObjectList)
	crdSubGroup.GET("/:object",apiHandler.handleGetCustomResourceObjectDetail)
	crdSubGroup.GET("/:object/event",apiHandler.handleGetCustomResourceObjectEvents)

	storageclassGroup := r.Group("/storageclass")
	storageclassGroup.GET("/",apiHandler.handleGetStorageClassList)
	storageclassGroup.GET("/:storageclass",apiHandler.handleGetStorageClass)
	storageclassGroup.GET("/:storageclass/persistentvolume",apiHandler.handleGetStorageClassPersistentVolumes)
	
	logGroup := r.Group("/log")
	logGroup.GET("/pod/:namespace/:pod", apiHandler.handleLogs)
	logGroup.GET("/pod/:namespace/:pod/:container", apiHandler.handleLogs)
	logGroup.GET("/resource/:namespace/:resourceName/:resourceType", apiHandler.handleLogSource)
	logGroup.GET("/file/:namespace/:pod/:container", apiHandler.handleLogFile)

	return nil
}

func (apiHandler *APIHandler) handleGetClusterRoleList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := clusterrole.GetClusterRoleList(k8sClient, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetClusterRoleDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("name")
	result, err := clusterrole.GetClusterRoleDetail(k8sClient, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetClusterRoleBindingList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := clusterrolebinding.GetClusterRoleBindingList(k8sClient, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetClusterRoleBindingDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("name")
	result, err := clusterrolebinding.GetClusterRoleBindingDetail(k8sClient, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetRoleList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := role.GetRoleList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetRoleDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	result, err := role.GetRoleDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetRoleBindingList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := rolebinding.GetRoleBindingList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetRoleBindingDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	result, err := rolebinding.GetRoleBindingDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCsrfToken(c *gin.Context) {
	action := c.Param("action")
	token := xsrftoken.Generate(apiHandler.cManager.CSRFKey(), "none", action)
	httphelper.RestfullResponse(c,http.StatusOK, api.CsrfToken{Token: token})
}

func (apiHandler *APIHandler) handleGetStatefulSetList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := statefulset.GetStatefulSetList(k8sClient, namespace, dataSelect,
		apiHandler.iManager.Metric().Client())
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetStatefulSetDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("statefulset")
	result, err := statefulset.GetStatefulSetDetail(k8sClient, apiHandler.iManager.Metric().Client(), namespace, name)

	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetStatefulSetPods(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("statefulset")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := statefulset.GetStatefulSetPods(k8sClient, apiHandler.iManager.Metric().Client(), dataSelect, name, namespace)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetStatefulSetEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("statefulset")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := event.GetResourceEvents(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetServiceList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := resourceService.GetServiceList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetServiceDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("service")
	result, err := resourceService.GetServiceDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetServiceEvent(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("service")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := resourceService.GetServiceEvents(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetServiceAccountList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := serviceaccount.GetServiceAccountList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetServiceAccountDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("serviceaccount")
	result, err := serviceaccount.GetServiceAccountDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetServiceAccountImagePullSecrets(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("serviceaccount")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := serviceaccount.GetServiceAccountImagePullSecrets(k8sClient, namespace, name, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetServiceAccountSecrets(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("serviceaccount")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := serviceaccount.GetServiceAccountSecrets(k8sClient, namespace, name, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetIngressDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	result, err := ingress.GetIngressDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetIngressList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	namespace := parseNamespacePathParameter(c)
	result, err := ingress.GetIngressList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetServicePods(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("service")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := resourceService.GetServicePods(k8sClient, apiHandler.iManager.Metric().Client(), namespace, name, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetNetworkPolicyList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := networkpolicy.GetNetworkPolicyList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetNetworkPolicyDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("networkpolicy")
	result, err := networkpolicy.GetNetworkPolicyDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetNodeList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := node.GetNodeList(k8sClient, dataSelect, apiHandler.iManager.Metric().Client())
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetNodeDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("name")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := node.GetNodeDetail(k8sClient, apiHandler.iManager.Metric().Client(), name, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetNodeEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("name")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := event.GetNodeEvents(k8sClient, dataSelect, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetNodePods(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("name")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := node.GetNodePods(k8sClient, apiHandler.iManager.Metric().Client(), dataSelect, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleDeploy(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	appDeploymentSpec := new(deployment.AppDeploymentSpec)
	if err := httphelper.ReadRequestBody(c,appDeploymentSpec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	if err := deployment.DeployApp(appDeploymentSpec, k8sClient); err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusCreated, appDeploymentSpec)
}

func (apiHandler *APIHandler) handleScaleResource(c *gin.Context) {
	cfg, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Query("namespace")
	kind := c.Param("kind")
	name := c.Param("name")
	count := c.Param("scaleBy")
	replicaCountSpec, err := scaling.ScaleResource(cfg, kind, namespace, name, count)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, replicaCountSpec)
}

func (apiHandler *APIHandler) handleGetReplicaCount(c *gin.Context) {
	cfg, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	kind := c.Param("kind")
	name := c.Param("name")
	replicaCounts, err := scaling.GetReplicaCounts(cfg, kind, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, replicaCounts)
}

func (apiHandler *APIHandler) handleDeployFromFile(c *gin.Context) {
	cfg, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	deploymentSpec := new(deployment.AppDeploymentFromFileSpec)
	if err := httphelper.ReadRequestBody(c,deploymentSpec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	isDeployed, err := deployment.DeployAppFromFile(cfg, deploymentSpec)
	if !isDeployed {
		errors.HandleInternalError(c, err)
		return
	}

	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}

	httphelper.RestfullResponse(c,http.StatusCreated, deployment.AppDeploymentFromFileResponse{
		Name:    deploymentSpec.Name,
		Content: deploymentSpec.Content,
		Error:   errorMessage,
	})
}

func (apiHandler *APIHandler) handleNameValidity(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	spec := new(validation.AppNameValiditySpec)
	if err := httphelper.ReadRequestBody(c,spec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	validity, err := validation.ValidateAppName(spec, k8sClient)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, validity)
}

func (apiHandler *APIHandler) handleImageReferenceValidity(c *gin.Context) {
	spec := new(validation.ImageReferenceValiditySpec)
	if err := httphelper.ReadRequestBody(c,spec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	validity, err := validation.ValidateImageReference(spec)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, validity)
}

func (apiHandler *APIHandler) handleProtocolValidity(c *gin.Context) {
	spec := new(validation.ProtocolValiditySpec)
	if err := httphelper.ReadRequestBody(c,spec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, validation.ValidateProtocol(spec))
}

func (apiHandler *APIHandler) handleGetAvailableProtocols(c *gin.Context) {
	httphelper.RestfullResponse(c,http.StatusOK, deployment.GetAvailableProtocols())
}

func (apiHandler *APIHandler) handleGetReplicationControllerList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := replicationcontroller.GetReplicationControllerList(k8sClient, namespace, dataSelect, apiHandler.iManager.Metric().Client())
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetReplicaSets(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := replicaset.GetReplicaSetList(k8sClient, namespace, dataSelect, apiHandler.iManager.Metric().Client())
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetReplicaSetDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	replicaSet := c.Param("replicaSet")
	result, err := replicaset.GetReplicaSetDetail(k8sClient, apiHandler.iManager.Metric().Client(), namespace, replicaSet)

	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetReplicaSetPods(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	replicaSet := c.Param("replicaSet")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := replicaset.GetReplicaSetPods(k8sClient, apiHandler.iManager.Metric().Client(), dataSelect, replicaSet, namespace)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetReplicaSetServices(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	replicaSet := c.Param("replicaSet")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := replicaset.GetReplicaSetServices(k8sClient, dataSelect, namespace, replicaSet)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetReplicaSetEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("replicaSet")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := event.GetResourceEvents(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)

}

func (apiHandler *APIHandler) handleGetPodEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	log.Println("Getting events related to a pod in namespace")
	namespace := c.Param("namespace")
	name := c.Param("pod")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := pod.GetEventsForPod(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

// Handles execute shell API call
func (apiHandler *APIHandler) handleExecShell(c *gin.Context) {
	sessionID, err := genTerminalSessionId()
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	cfg, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	terminalSessions.Set(sessionID, TerminalSession{
		id:       sessionID,
		bound:    make(chan error),
		sizeChan: make(chan remotecommand.TerminalSize),
	})
	go WaitForTerminal(k8sClient, cfg, c, sessionID)
	httphelper.RestfullResponse(c,http.StatusOK, TerminalResponse{ID: sessionID})
}

func (apiHandler *APIHandler) handleGetDeployments(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := deployment.GetDeploymentList(k8sClient, namespace, dataSelect, apiHandler.iManager.Metric().Client())
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetDeploymentDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("deployment")
	result, err := deployment.GetDeploymentDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetDeploymentEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("deployment")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := event.GetResourceEvents(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetDeploymentOldReplicaSets(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("deployment")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := deployment.GetDeploymentOldReplicaSets(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetDeploymentNewReplicaSet(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("deployment")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := deployment.GetDeploymentNewReplicaSet(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetPods(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics // download standard metrics - cpu, and memory - by default
	result, err := pod.GetPodList(k8sClient, apiHandler.iManager.Metric().Client(), namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetPodDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("pod")
	result, err := pod.GetPodDetail(k8sClient, apiHandler.iManager.Metric().Client(), namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetReplicationControllerDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("replicationController")
	result, err := replicationcontroller.GetReplicationControllerDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleUpdateReplicasCount(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("replicationController")
	spec := new(replicationcontroller.ReplicationControllerSpec)
	if err := httphelper.ReadRequestBody(c,spec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	if err := replicationcontroller.UpdateReplicasCount(k8sClient, namespace, name, spec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusAccepted, nil)
}

func (apiHandler *APIHandler) handleGetResource(c *gin.Context) {
	config, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	verber, err := apiHandler.cManager.VerberClient(c, config)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	kind := c.Param("kind")
	namespace := c.Param("namespace")
	name := c.Param("name")
	ok := namespace == ""
	result, err := verber.Get(kind, ok, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handlePutResource(
	c *gin.Context) {
	config, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	verber, err := apiHandler.cManager.VerberClient(c, config)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	kind := c.Param("kind")
	namespace := c.Param("namespace")
	name := c.Param("name")
	ok := namespace == ""
	putSpec := &runtime.Unknown{}
	if err := httphelper.ReadRequestBody(c,putSpec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	if err := verber.Put(kind, ok, namespace, name, putSpec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusCreated, nil)
}

func (apiHandler *APIHandler) handleDeleteResource(
	c *gin.Context) {
	config, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	verber, err := apiHandler.cManager.VerberClient(c, config)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	kind := c.Param("kind")
	namespace := c.Param("namespace")
	name := c.Param("name")
	ok := namespace == ""
	if err := verber.Delete(kind, ok, namespace, name); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	// Try to unpin resource if it was pinned.
	pinnedResource := &settingsApi.PinnedResource{
		Name:      name,
		Kind:      kind,
		Namespace: namespace,
	}

	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	if err = apiHandler.sManager.DeletePinnedResource(k8sClient, pinnedResource); err != nil {
		if !errors.IsNotFoundError(err) {
			log.Printf("error while unpinning resource: %s", err.Error())
		}
	}

	httphelper.RestfullResponse(c,http.StatusOK, nil)
}

func (apiHandler *APIHandler) handleGetReplicationControllerPods(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	rc := c.Param("replicationController")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := replicationcontroller.GetReplicationControllerPods(k8sClient, apiHandler.iManager.Metric().Client(), dataSelect, rc, namespace)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleCreateNamespace(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespaceSpec := new(ns.NamespaceSpec)
	if err := httphelper.ReadRequestBody(c,namespaceSpec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	if err := ns.CreateNamespace(namespaceSpec, k8sClient); err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusCreated, namespaceSpec)
}

func (apiHandler *APIHandler) handleGetNamespaces(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := ns.GetNamespaceList(k8sClient, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetNamespaceDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("name")
	result, err := ns.GetNamespaceDetail(k8sClient, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetNamespaceEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("name")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := event.GetNamespaceEvents(k8sClient, dataSelect, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleCreateImagePullSecret(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	spec := new(secret.ImagePullSecretSpec)
	if err := httphelper.ReadRequestBody(c,spec); err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	result, err := secret.CreateSecret(k8sClient, spec)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusCreated, result)
}

func (apiHandler *APIHandler) handleGetSecretDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	result, err := secret.GetSecretDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetSecretList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	namespace := parseNamespacePathParameter(c)
	result, err := secret.GetSecretList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetConfigMapList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := configmap.GetConfigMapList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetConfigMapDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("configmap")
	result, err := configmap.GetConfigMapDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetPersistentVolumeList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := persistentvolume.GetPersistentVolumeList(k8sClient, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetPersistentVolumeDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("persistentvolume")
	result, err := persistentvolume.GetPersistentVolumeDetail(k8sClient, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetPersistentVolumeClaimList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := persistentvolumeclaim.GetPersistentVolumeClaimList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetPersistentVolumeClaimDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	result, err := persistentvolumeclaim.GetPersistentVolumeClaimDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetPodContainers(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("pod")
	result, err := container.GetPodContainers(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetReplicationControllerEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("replicationController")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := event.GetResourceEvents(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetReplicationControllerServices(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("replicationController")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := replicationcontroller.GetReplicationControllerServices(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetDaemonSetList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := daemonset.GetDaemonSetList(k8sClient, namespace, dataSelect, apiHandler.iManager.Metric().Client())
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetDaemonSetDetail(
	c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("daemonSet")
	result, err := daemonset.GetDaemonSetDetail(k8sClient, apiHandler.iManager.Metric().Client(), namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetDaemonSetPods(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("daemonSet")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := daemonset.GetDaemonSetPods(k8sClient, apiHandler.iManager.Metric().Client(), dataSelect, name, namespace)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetDaemonSetServices(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	daemonSet := c.Param("daemonSet")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := daemonset.GetDaemonSetServices(k8sClient, dataSelect, namespace, daemonSet)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetDaemonSetEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("daemonSet")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := event.GetResourceEvents(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetHorizontalPodAutoscalerList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := horizontalpodautoscaler.GetHorizontalPodAutoscalerList(k8sClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetHorizontalPodAutoscalerListForResource(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
///:namespace/:horizontalpodautoscaler/:kind
	namespace := c.Param("namespace")
	name := c.Param("horizontalpodautoscaler")
	kind := c.Param("kind")
	result, err := horizontalpodautoscaler.GetHorizontalPodAutoscalerListForResource(k8sClient, namespace, kind, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetHorizontalPodAutoscalerDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("horizontalpodautoscaler")
	result, err := horizontalpodautoscaler.GetHorizontalPodAutoscalerDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetJobList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := job.GetJobList(k8sClient, namespace, dataSelect, apiHandler.iManager.Metric().Client())
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetJobDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	result, err := job.GetJobDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetJobPods(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := job.GetJobPods(k8sClient, apiHandler.iManager.Metric().Client(), dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetJobEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := job.GetJobEvents(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCronJobList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := cronjob.GetCronJobList(k8sClient, namespace, dataSelect, apiHandler.iManager.Metric().Client())
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCronJobDetail(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	result, err := cronjob.GetCronJobDetail(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCronJobJobs(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	active := true
	if c.Query("active") == "false" {
		active = false
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := cronjob.GetCronJobJobs(k8sClient, apiHandler.iManager.Metric().Client(), dataSelect, namespace, name, active)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCronJobEvents(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := cronjob.GetCronJobEvents(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleTriggerCronJob(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	err = cronjob.TriggerCronJob(k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, nil)
}

func (apiHandler *APIHandler) handleGetStorageClassList(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := storageclass.GetStorageClassList(k8sClient, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetStorageClass(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("storageclass")
	result, err := storageclass.GetStorageClass(k8sClient, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetStorageClassPersistentVolumes(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("storageclass")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := persistentvolume.GetStorageClassPersistentVolumes(k8sClient,
		name, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetPodPersistentVolumeClaims(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("pod")
	namespace := c.Param("namespace")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := persistentvolumeclaim.GetPodPersistentVolumeClaims(k8sClient,
		namespace, name, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCustomResourceDefinitionList(c *gin.Context) {
	apiextensionsclient, err := apiHandler.cManager.APIExtensionsClient(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := customresourcedefinition.GetCustomResourceDefinitionList(apiextensionsclient, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCustomResourceDefinitionDetail(c *gin.Context) {
	config, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	apiextensionsclient, err := apiHandler.cManager.APIExtensionsClient(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("crd")
	result, err := customresourcedefinition.GetCustomResourceDefinitionDetail(apiextensionsclient, config, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCustomResourceObjectList(c *gin.Context) {
	config, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	apiextensionsclient, err := apiHandler.cManager.APIExtensionsClient(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	crdName := c.Param("crd")
	namespace := parseNamespacePathParameter(c)
	dataSelect := parser.ParseDataSelectPathParameter(c)
	result, err := customresourcedefinition.GetCustomResourceObjectList(apiextensionsclient, config, namespace, dataSelect, crdName)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCustomResourceObjectDetail(c *gin.Context) {
	config, err := apiHandler.cManager.Config(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	apiextensionsclient, err := apiHandler.cManager.APIExtensionsClient(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("object")
	crdName := c.Param("crd")
	namespace := parseNamespacePathParameter(c)
	result, err := customresourcedefinition.GetCustomResourceObjectDetail(apiextensionsclient, namespace, config, crdName, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleGetCustomResourceObjectEvents(c *gin.Context) {
	log.Println("Getting events related to a custom resource object in namespace")

	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	name := c.Param("object")
	namespace := c.Param("namespace")
	dataSelect := parser.ParseDataSelectPathParameter(c)
	dataSelect.MetricQuery = dataselect.StandardMetrics
	result, err := customresourcedefinition.GetEventsForCustomResourceObject(k8sClient, dataSelect, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleLogSource(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	resourceName := c.Param("resourceName")
	resourceType := c.Param("resourceType")
	namespace := c.Param("namespace")
	logSources, err := logs.GetLogSources(k8sClient, namespace, resourceName, resourceType)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, logSources)
}

func (apiHandler *APIHandler) handleLogs(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	namespace := c.Param("namespace")
	podID := c.Param("pod")
	containerID := c.Param("container")

	refTimestamp := c.Query("referenceTimestamp")
	if refTimestamp == "" {
		refTimestamp = logs.NewestTimestamp
	}

	refLineNum, err := strconv.Atoi(c.Query("referenceLineNum"))
	if err != nil {
		refLineNum = 0
	}
	usePreviousLogs := c.Query("previous") == "true"
	offsetFrom, err1 := strconv.Atoi(c.Query("offsetFrom"))
	offsetTo, err2 := strconv.Atoi(c.Query("offsetTo"))
	logFilePosition := c.Query("logFilePosition")

	logSelector := logs.DefaultSelection
	if err1 == nil && err2 == nil {
		logSelector = &logs.Selection{
			ReferencePoint: logs.LogLineId{
				LogTimestamp: logs.LogTimestamp(refTimestamp),
				LineNum:      refLineNum,
			},
			OffsetFrom:      offsetFrom,
			OffsetTo:        offsetTo,
			LogFilePosition: logFilePosition,
		}
	}

	result, err := container.GetLogDetails(k8sClient, namespace, podID, containerID, logSelector, usePreviousLogs)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c,http.StatusOK, result)
}

func (apiHandler *APIHandler) handleLogFile(c *gin.Context) {
	k8sClient, err := apiHandler.cManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	namespace := c.Param("namespace")
	podID := c.Param("pod")
	containerID := c.Param("container")
	usePreviousLogs := c.Query("previous") == "true"

	logStream, err := container.GetLogFile(k8sClient, namespace, podID, containerID, usePreviousLogs)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	handleDownload(c, logStream)
}

// parseNamespacePathParameter parses namespace selector for list pages in path parameter.
// The namespace selector is a comma separated list of namespaces that are trimmed.
// No namespaces means "view all user namespaces", i.e., everything except kube-system.
func parseNamespacePathParameter(c *gin.Context) *common.NamespaceQuery {
	namespace := c.Param("namespace")
	namespaces := strings.Split(namespace, ",")
	var nonEmptyNamespaces []string
	for _, n := range namespaces {
		n = strings.Trim(n, " ")
		if len(n) > 0 {
			nonEmptyNamespaces = append(nonEmptyNamespaces, n)
		}
	}
	return common.NewNamespaceQuery(nonEmptyNamespaces)
}

// parseNamespacePathParameter parses namespace selector for list pages in path parameter.
// The namespace selector is a comma separated list of namespaces that are trimmed.
// No namespaces means "view all user namespaces", i.e., everything except kube-system.
func parseNamespaceQueryParameter(c *gin.Context) *common.NamespaceQuery {
	namespace := c.Query("namespace")
	namespaces := strings.Split(namespace, ",")
	var nonEmptyNamespaces []string
	for _, n := range namespaces {
		n = strings.Trim(n, " ")
		if len(n) > 0 {
			nonEmptyNamespaces = append(nonEmptyNamespaces, n)
		}
	}
	return common.NewNamespaceQuery(nonEmptyNamespaces)
}
