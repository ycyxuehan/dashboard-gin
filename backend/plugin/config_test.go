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

package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ycyxuehan/dashboard-gin/backend/plugin/apis/v1alpha1"

	authApi "github.com/ycyxuehan/dashboard-gin/backend/auth/api"
	clientapi "github.com/ycyxuehan/dashboard-gin/backend/client/api"
	"github.com/ycyxuehan/dashboard-gin/backend/plugin/client/clientset/versioned"
	fakePluginClientset "github.com/ycyxuehan/dashboard-gin/backend/plugin/client/clientset/versioned/fake"
	v1 "k8s.io/api/authorization/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fakeK8sClient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type httpWriter struct{

}


func Test_handleConfig(t *testing.T) {
	ns := "default"
	pluginName := "test-plugin"
	filename := "plugin-test.js"
	cfgMapName := "plugin-test-cfgMap"
	h := Handler{&fakeClientManager{}}

	pcs, _ := h.cManager.PluginClient(nil)
	_, _ = pcs.DashboardV1alpha1().Plugins(ns).Create(&v1alpha1.Plugin{
		ObjectMeta: metaV1.ObjectMeta{Name: pluginName, Namespace: ns},
		Spec: v1alpha1.PluginSpec{
			Source: v1alpha1.Source{
				ConfigMapRef: &coreV1.ConfigMapEnvSource{
					LocalObjectReference: coreV1.LocalObjectReference{Name: cfgMapName},
				},
				Filename: filename}},
	}, metaV1.CreateOptions{})


	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(http.MethodGet, "/api/v1/plugin/config", nil)
	h.handleConfig(c)
}

type fakeClientManager struct {
	k8sClient    kubernetes.Interface
	pluginClient versioned.Interface
}

func (cm *fakeClientManager) Client(c *gin.Context) (kubernetes.Interface, error) {
	panic("implement me")
}

func (cm *fakeClientManager) InsecureClient() kubernetes.Interface {
	if cm.k8sClient == nil {
		cm.k8sClient = fakeK8sClient.NewSimpleClientset()
	}
	return cm.k8sClient
}

func (cm *fakeClientManager) APIExtensionsClient(c *gin.Context) (clientset.Interface, error) {
	panic("implement me")
}

func (cm *fakeClientManager) PluginClient(c *gin.Context) (versioned.Interface, error) {
	if cm.pluginClient == nil {
		cm.pluginClient = fakePluginClientset.NewSimpleClientset()
	}
	return cm.pluginClient, nil
}

func (cm *fakeClientManager) InsecureAPIExtensionsClient() clientset.Interface {
	panic("implement me")
}

func (cm *fakeClientManager) InsecurePluginClient() versioned.Interface {
	if cm.pluginClient == nil {
		cm.pluginClient = fakePluginClientset.NewSimpleClientset()
	}
	return cm.pluginClient
}

func (cm *fakeClientManager) CanI(c *gin.Context, ssar *v1.SelfSubjectAccessReview) bool {
	panic("implement me")
}

func (cm *fakeClientManager) Config(c *gin.Context) (*rest.Config, error) {
	panic("implement me")
}

func (cm *fakeClientManager) ClientCmdConfig(c *gin.Context) (clientcmd.ClientConfig, error) {
	panic("implement me")
}

func (cm *fakeClientManager) CSRFKey() string {
	panic("implement me")
}

func (cm *fakeClientManager) HasAccess(authInfo api.AuthInfo) error {
	panic("implement me")
}

func (cm *fakeClientManager) VerberClient(c *gin.Context, config *rest.Config) (clientapi.ResourceVerber, error) {
	panic("implement me")
}

func (cm *fakeClientManager) SetTokenManager(manager authApi.TokenManager) {
	panic("implement me")
}
