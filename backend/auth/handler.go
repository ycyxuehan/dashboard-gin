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

package auth

import (
	"github.com/gin-gonic/gin"
	authApi "github.com/ycyxuehan/dashboard-gin/backend/auth/api"
	"github.com/ycyxuehan/dashboard-gin/backend/errors"
	"github.com/ycyxuehan/dashboard-gin/backend/validation"
	"github.com/ycyxuehan/dashboard-gin/backend/utils/httphelper"
)

// AuthHandler manages all endpoints related to dashboard auth, such as login.
type AuthHandler struct {
	manager authApi.AuthManager
}

// Install creates new endpoints for dashboard auth, such as login. It allows user to log in to dashboard using
// one of the supported methods. See AuthManager and Authenticator for more information.
func (a AuthHandler) Install(router *gin.RouterGroup) {
	router.POST("/login", a.handleLogin)
	router.GET("/login/status", a.handleLoginStatus)
	router.POST("/token/refresh", a.handleJWETokenRefresh)
	router.GET("/login/modes", a.handleLoginModes)
	router.GET("/login/skippable", a.handleLoginSkippable)
}

func (a AuthHandler) handleLogin(c *gin.Context) {
	loginSpec := new(authApi.LoginSpec)
	if err := httphelper.ReadRequestBody(c, &loginSpec); err != nil {
		httphelper.RestfullResponse(c, errors.HandleHTTPError(err), err)
		return
	}

	loginResponse, err := a.manager.Login(loginSpec)
	if err != nil {
		httphelper.RestfullResponse(c, errors.HandleHTTPError(err), err)
		return
	}

	httphelper.RestfullResponse(c, httphelper.SUCCESS, loginResponse)
}

func (a *AuthHandler) handleLoginStatus(c *gin.Context) {
	httphelper.RestfullResponse(c, httphelper.SUCCESS,  validation.ValidateLoginStatus(c))
}

func (a *AuthHandler) handleJWETokenRefresh(c *gin.Context) {
	tokenRefreshSpec := new(authApi.TokenRefreshSpec)
	if err := httphelper.ReadRequestBody(c, &tokenRefreshSpec); err != nil {
		httphelper.RestfullResponse(c, errors.HandleHTTPError(err), err)
		return
	}

	refreshedJWEToken, err := a.manager.Refresh(tokenRefreshSpec.JWEToken)
	if err != nil {
		httphelper.RestfullResponse(c, errors.HandleHTTPError(err), err)
		return
	}

	httphelper.RestfullResponse(c,  httphelper.SUCCESS,  &authApi.AuthResponse{
		JWEToken: refreshedJWEToken,
		Errors:   make([]error, 0),
	})
}

func (a *AuthHandler) handleLoginModes(c *gin.Context) {
	httphelper.RestfullResponse(c,  httphelper.SUCCESS, authApi.LoginModesResponse{Modes: a.manager.AuthenticationModes()})
}

func (a *AuthHandler) handleLoginSkippable(c *gin.Context) {
	httphelper.RestfullResponse(c,  httphelper.SUCCESS, authApi.LoginSkippableResponse{Skippable: a.manager.AuthenticationSkippable()})
}

// NewAuthHandler created AuthHandler instance.
func NewAuthHandler(manager authApi.AuthManager) AuthHandler {
	return AuthHandler{manager: manager}
}
