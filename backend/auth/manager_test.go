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
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	authApi "github.com/ycyxuehan/dashboard-gin/backend/auth/api"
	"github.com/ycyxuehan/dashboard-gin/backend/client"
	clientapi "github.com/ycyxuehan/dashboard-gin/backend/client/api"
	"github.com/ycyxuehan/dashboard-gin/backend/errors"

	pluginclientset "github.com/ycyxuehan/dashboard-gin/backend/plugin/client/clientset/versioned"
	v1 "k8s.io/api/authorization/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func areErrorsEqual(err1, err2 error) bool {
	return (err1 != nil && err2 != nil && err1.Error() == err2.Error()) ||
		(err1 == nil && err2 == nil)
}

type fakeClientManager struct {
	HasAccessError error
}

func (fcm *fakeClientManager) Client(c *gin.Context) (kubernetes.Interface, error) {
	return nil, nil
}

func (fcm *fakeClientManager) APIExtensionsClient(c *gin.Context) (apiextensionsclientset.Interface, error) {
	return nil, nil
}

func (fcm *fakeClientManager) PluginClient(c *gin.Context) (pluginclientset.Interface, error) {
	return nil, nil
}

func (fcm *fakeClientManager) InsecureClient() kubernetes.Interface {
	return nil
}

func (fcm *fakeClientManager) InsecureAPIExtensionsClient() apiextensionsclientset.Interface {
	return nil
}

func (fcm *fakeClientManager) InsecurePluginClient() pluginclientset.Interface {
	return nil
}

func (fcm *fakeClientManager) SetTokenManager(manager authApi.TokenManager) {}

func (fcm *fakeClientManager) Config(c *gin.Context) (*rest.Config, error) {
	return nil, nil
}

func (fcm *fakeClientManager) ClientCmdConfig(c *gin.Context) (clientcmd.ClientConfig, error) {
	return clientcmd.NewDefaultClientConfig(api.Config{}, &clientcmd.ConfigOverrides{}), nil
}

func (fcm *fakeClientManager) CSRFKey() string {
	return ""
}

func (fcm *fakeClientManager) HasAccess(authInfo api.AuthInfo) error {
	return fcm.HasAccessError
}

func (fcm *fakeClientManager) VerberClient(c *gin.Context, config *rest.Config) (clientapi.ResourceVerber, error) {
	return client.NewResourceVerber(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil), nil
}

func (fcm *fakeClientManager) CanI(c *gin.Context, ssar *v1.SelfSubjectAccessReview) bool {
	return true
}

type fakeTokenManager struct {
	GeneratedToken string
	Error          error
}

func (ftm *fakeTokenManager) Refresh(string) (string, error) {
	return "", nil
}

func (ftm *fakeTokenManager) SetTokenTTL(time.Duration) {}

func (ftm *fakeTokenManager) Generate(authInfo api.AuthInfo) (string, error) {
	return ftm.GeneratedToken, ftm.Error
}

func (ftm *fakeTokenManager) Decrypt(jweToken string) (*api.AuthInfo, error) {
	return nil, nil
}

func TestAuthManager_Login(t *testing.T) {
	unauthorizedErr := errors.NewUnauthorized("Unauthorized")

	cases := []struct {
		info        string
		spec        *authApi.LoginSpec
		cManager    clientapi.ClientManager
		tManager    authApi.TokenManager
		expected    *authApi.AuthResponse
		expectedErr error
	}{
		{
			"Empty login spec should throw authenticator error",
			&authApi.LoginSpec{},
			&fakeClientManager{HasAccessError: nil},
			&fakeTokenManager{},
			nil,
			errors.NewInvalid("Not enough data to create authenticator."),
		}, {
			"Not recognized token should throw unauthorized error",
			&authApi.LoginSpec{Token: "not-existing-token"},
			&fakeClientManager{HasAccessError: unauthorizedErr},
			&fakeTokenManager{},
			&authApi.AuthResponse{Errors: []error{unauthorizedErr}},
			nil,
		}, {
			"Recognized token should allow login and return JWE token",
			&authApi.LoginSpec{Token: "existing-token"},
			&fakeClientManager{HasAccessError: nil},
			&fakeTokenManager{GeneratedToken: "generated-token"},
			&authApi.AuthResponse{JWEToken: "generated-token", Errors: make([]error, 0)},
			nil,
		}, {
			"Should propagate error on unexpected error",
			&authApi.LoginSpec{Token: "test-token"},
			&fakeClientManager{HasAccessError: errors.NewInvalid("Unexpected error")},
			&fakeTokenManager{},
			&authApi.AuthResponse{Errors: make([]error, 0)},
			errors.NewInvalid("Unexpected error"),
		},
	}

	for _, c := range cases {
		authManager := NewAuthManager(c.cManager, c.tManager, authApi.AuthenticationModes{authApi.Token: true}, true)
		response, err := authManager.Login(c.spec)

		if !areErrorsEqual(err, c.expectedErr) {
			t.Errorf("Test Case: %s. Expected error to be: %v, but got %v.",
				c.info, c.expectedErr, err)
		}

		if !reflect.DeepEqual(response, c.expected) {
			t.Errorf("Test Case: %s. Expected response to be: %v, but got %v.",
				c.info, c.expected, response)
		}
	}
}

func TestAuthManager_AuthenticationModes(t *testing.T) {
	cManager := &fakeClientManager{}
	tManager := &fakeTokenManager{}
	cases := []struct {
		modes    authApi.AuthenticationModes
		expected []authApi.AuthenticationMode
	}{
		{authApi.AuthenticationModes{}, []authApi.AuthenticationMode{}},
		{authApi.AuthenticationModes{authApi.Token: true}, []authApi.AuthenticationMode{authApi.Token}},
	}

	for _, c := range cases {
		authManager := NewAuthManager(cManager, tManager, c.modes, true)
		got := authManager.AuthenticationModes()

		if !reflect.DeepEqual(got, c.expected) {
			t.Errorf("Expected %v, but got %v.", c.expected, got)
		}
	}
}

func TestAuthManager_AuthenticationSkippable(t *testing.T) {
	cManager := &fakeClientManager{}
	tManager := &fakeTokenManager{}
	cModes := authApi.AuthenticationModes{}

	for _, flag := range []bool{true, false} {
		authManager := NewAuthManager(cManager, tManager, cModes, flag)
		got := authManager.AuthenticationSkippable()
		if got != flag {
			t.Errorf("Expected %v, but got %v.", flag, got)
		}
	}
}
