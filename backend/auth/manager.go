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
	"k8s.io/client-go/tools/clientcmd/api"

	authApi "github.com/ycyxuehan/dashboard-gin/backend/auth/api"
	clientapi "github.com/ycyxuehan/dashboard-gin/backend/client/api"
	"github.com/ycyxuehan/dashboard-gin/backend/errors"
)

// Implements AuthManager interface
type authManager struct {
	tokenManager            authApi.TokenManager
	clientManager           clientapi.ClientManager
	authenticationModes     authApi.AuthenticationModes
	authenticationSkippable bool
}

// Login implements auth manager. See AuthManager interface for more information.
func (am authManager) Login(spec *authApi.LoginSpec) (*authApi.AuthResponse, error) {
	authenticator, err := am.getAuthenticator(spec)
	if err != nil {
		return nil, err
	}

	authInfo, err := authenticator.GetAuthInfo()
	if err != nil {
		return nil, err
	}

	err = am.healthCheck(authInfo)
	nonCriticalErrors, criticalError := errors.HandleError(err)
	if criticalError != nil || len(nonCriticalErrors) > 0 {
		return &authApi.AuthResponse{Errors: nonCriticalErrors}, criticalError
	}

	token, err := am.tokenManager.Generate(authInfo)
	if err != nil {
		return nil, err
	}

	return &authApi.AuthResponse{JWEToken: token, Errors: nonCriticalErrors}, nil
}

// Refresh implements auth manager. See AuthManager interface for more information.
func (am authManager) Refresh(jweToken string) (string, error) {
	return am.tokenManager.Refresh(jweToken)
}

func (am authManager) AuthenticationModes() []authApi.AuthenticationMode {
	return am.authenticationModes.Array()
}

func (am authManager) AuthenticationSkippable() bool {
	return am.authenticationSkippable
}

// Returns authenticator based on provided LoginSpec.
func (am authManager) getAuthenticator(spec *authApi.LoginSpec) (authApi.Authenticator, error) {
	if len(am.authenticationModes) == 0 {
		return nil, errors.NewInvalid("All authentication options disabled. Check --authentication-modes argument for more information.")
	}

	switch {
	case len(spec.Token) > 0 && am.authenticationModes.IsEnabled(authApi.Token):
		return NewTokenAuthenticator(spec), nil
	case len(spec.Username) > 0 && len(spec.Password) > 0 && am.authenticationModes.IsEnabled(authApi.Basic):
		return NewBasicAuthenticator(spec), nil
	case len(spec.KubeConfig) > 0:
		return NewKubeConfigAuthenticator(spec, am.authenticationModes), nil
	}

	return nil, errors.NewInvalid("Not enough data to create authenticator.")
}

// Checks if user data extracted from provided AuthInfo structure is valid and user is correctly authenticated
// by K8S apiserver.
func (am authManager) healthCheck(authInfo api.AuthInfo) error {
	return am.clientManager.HasAccess(authInfo)
}

// NewAuthManager creates auth manager.
func NewAuthManager(clientManager clientapi.ClientManager, tokenManager authApi.TokenManager,
	authenticationModes authApi.AuthenticationModes, authenticationSkippable bool) authApi.AuthManager {
	return &authManager{
		tokenManager:            tokenManager,
		clientManager:           clientManager,
		authenticationModes:     authenticationModes,
		authenticationSkippable: authenticationSkippable,
	}
}
