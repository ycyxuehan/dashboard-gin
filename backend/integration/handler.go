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

package integration

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ycyxuehan/dashboard-gin/backend/integration/api"
	"github.com/ycyxuehan/dashboard-gin/backend/utils/httphelper"
)

// IntegrationHandler manages all endpoints related to integrated applications, such as state.
type IntegrationHandler struct {
	manager IntegrationManager
}

// Install creates new endpoints for integrations. All information that any integration would want
// to expose by creating new endpoints should be kept here, i.e. helm integration might want to
// create endpoint to list available releases/charts.
//
// By default endpoint for checking state of the integrations is installed. It allows user
// to check state of integration by accessing `<DASHBOARD_URL>/api/v1/integration/{name}/state`.
func (self IntegrationHandler) Install(ws *gin.RouterGroup) {
	ws.GET("/integreation/:name/state", self.handleGetState)
}

func (self IntegrationHandler) handleGetState(c *gin.Context) {
	integrationName := c.Param("name")
	state, err := self.manager.GetState(api.IntegrationID(integrationName))
	if err != nil {
		httphelper.RestfullResponse(c, http.StatusInternalServerError, err)
		return
	}
	httphelper.RestfullResponse(c, 0, state)
}

// NewIntegrationHandler creates IntegrationHandler.
func NewIntegrationHandler(manager IntegrationManager) IntegrationHandler {
	return IntegrationHandler{manager: manager}
}
