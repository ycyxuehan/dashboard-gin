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
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ycyxuehan/dashboard-gin/backend/handler/parser"
	"github.com/ycyxuehan/dashboard-gin/backend/utils/httphelper"

	clientapi "github.com/ycyxuehan/dashboard-gin/backend/client/api"
	"github.com/ycyxuehan/dashboard-gin/backend/errors"
)

const (
	contentTypeHeader = "Content-Type"
	jsContentType     = "text/javascript; charset=utf-8"
)

// Handler manages all endpoints related to plugin use cases, such as list and get.
type Handler struct {
	cManager clientapi.ClientManager
}

// Install creates new endpoints for plugins. All information that any plugin would want
// to expose by creating new endpoints should be kept here, i.e. plugin service might want to
// create endpoint to list available proxy paths to another backend.
//
// By default, endpoint for getting and listing plugins is installed. It allows user
// to list the installed plugins and get the source code for a plugin.
func (h *Handler) Install(r *gin.RouterGroup) {
	g := r.Group("/plugin")
	g.GET("/config", h.handleConfig)
	g.GET("/:namespace", h.handlePluginList)
	g.GET("/:namespace/:pluginName", h.servePluginSource)
}

// NewPluginHandler creates plugin.Handler.
func NewPluginHandler(cManager clientapi.ClientManager) *Handler {
	return &Handler{cManager: cManager}
}

func (h *Handler) handlePluginList(c *gin.Context) {
	pluginClient, err := h.cManager.PluginClient(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	namespace := c.Param("namespace")
	dataSelect := parser.ParseDataSelectPathParameter(c)

	result, err := GetPluginList(pluginClient, namespace, dataSelect)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c, httphelper.SUCCESS, &result)
}

func (h *Handler) servePluginSource(c *gin.Context) {
	// TODO: Change these to secure clients once SystemJS can send proper auth headers.
	pluginClient := h.cManager.InsecurePluginClient()
	k8sClient := h.cManager.InsecureClient()

	namespace := c.Param("namespace")
	// Removes .js extension if it's present
	pluginName := c.Param("pluginName")
	name := strings.TrimSuffix(pluginName, filepath.Ext(pluginName))

	result, err := GetPluginSource(pluginClient, k8sClient, namespace, name)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c, httphelper.SUCCESS, &result)
}
