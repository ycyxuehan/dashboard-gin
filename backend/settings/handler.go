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

package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ycyxuehan/dashboard-gin/backend/args"
	clientapi "github.com/ycyxuehan/dashboard-gin/backend/client/api"
	"github.com/ycyxuehan/dashboard-gin/backend/errors"
	"github.com/ycyxuehan/dashboard-gin/backend/settings/api"
	"github.com/ycyxuehan/dashboard-gin/backend/utils/httphelper"
)

// SettingsHandler manages all endpoints related to settings management.
type SettingsHandler struct {
	manager       api.SettingsManager
	clientManager clientapi.ClientManager
}

// Install creates new endpoints for settings management.
func (s *SettingsHandler) Install(r *gin.RouterGroup) {
	settings := r.Group("/settings")
	global := settings.Group("/global")
	global.GET("/", s.handleSettingsGlobalGet)
	global.GET("/cani", s.handleSettingsGlobalCanI)
	global.PUT("/", s.handleSettingsGlobalSave)

	pinner:=settings.Group("/pinner")
	pinner.GET("/", s.handleSettingsGetPinned)
	pinner.PUT("/", s.handleSettingsSavePinned)
	pinner.GET("/cani", s.handleSettingsGlobalCanI)
	pinner.DELETE("/:kind/:name", s.handleSettingsDeletePinned)
	pinner.DELETE("/:kind/:name/:namespace", s.handleSettingsDeletePinned)
}

func (s *SettingsHandler) handleSettingsGlobalCanI(c *gin.Context) {
	verb := c.Query("verb")
	if len(verb) == 0 {
		verb = http.MethodGet
	}

	canI := s.clientManager.CanI(c, clientapi.ToSelfSubjectAccessReview(
		args.Holder.GetNamespace(),
		api.SettingsConfigMapName,
		api.ConfigMapKindName,
		verb,
	))

	if args.Holder.GetDisableSettingsAuthorizer() {
		canI = true
	}
	httphelper.RestfullResponse(c, httphelper.SUCCESS, clientapi.CanIResponse{Allowed: canI})
}

func (s *SettingsHandler) handleSettingsGlobalGet(c *gin.Context) {
	client := s.clientManager.InsecureClient()
	result := s.manager.GetGlobalSettings(client)
	httphelper.RestfullResponse(c, httphelper.SUCCESS, result)
}

func (s *SettingsHandler) handleSettingsGlobalSave(c *gin.Context) {
	settings := new(api.Settings)
	if err := httphelper.ReadRequestBody(c, settings); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	client, err := s.clientManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	if err := s.manager.SaveGlobalSettings(client, settings); err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c, httphelper.SUCCESS, settings) //http.StatusCreated
}

func (s *SettingsHandler) handleSettingsGetPinned(c *gin.Context) {
	client := s.clientManager.InsecureClient()
	result := s.manager.GetPinnedResources(client)
	httphelper.RestfullResponse(c, httphelper.SUCCESS, result)
}

func (s *SettingsHandler) handleSettingsSavePinned(c *gin.Context) {
	pinnedResource := new(api.PinnedResource)
	if err := httphelper.ReadRequestBody(c, pinnedResource); err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	client, err := s.clientManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	if err := s.manager.SavePinnedResource(client, pinnedResource); err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c, httphelper.SUCCESS, pinnedResource) //http.StatusCreated
}

func (s *SettingsHandler) handleSettingsDeletePinned(c *gin.Context) {
	pinnedResource := &api.PinnedResource{
		Kind:      c.Param("kind"),
		Name:      c.Param("name"),
		Namespace: c.Param("namespace"),
	}

	client, err := s.clientManager.Client(c)
	if err != nil {
		errors.HandleInternalError(c, err)
		return
	}

	if err := s.manager.DeletePinnedResource(client, pinnedResource); err != nil {
		errors.HandleInternalError(c, err)
		return
	}
	httphelper.RestfullResponse(c, http.StatusNoContent, nil) 
}

// NewSettingsHandler creates SettingsHandler.
func NewSettingsHandler(manager api.SettingsManager, clientManager clientapi.ClientManager) SettingsHandler {
	return SettingsHandler{manager: manager, clientManager: clientManager}
}
