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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ycyxuehan/dashboard-gin/backend/args"
	"github.com/ycyxuehan/dashboard-gin/backend/auth"
	authApi "github.com/ycyxuehan/dashboard-gin/backend/auth/api"
	"github.com/ycyxuehan/dashboard-gin/backend/auth/jwe"
	"github.com/ycyxuehan/dashboard-gin/backend/client"
	"github.com/ycyxuehan/dashboard-gin/backend/settings"
	"github.com/ycyxuehan/dashboard-gin/backend/sync"
	"github.com/ycyxuehan/dashboard-gin/backend/systembanner"
	"k8s.io/client-go/kubernetes/fake"
)

func getTokenManager() authApi.TokenManager {
	c := fake.NewSimpleClientset()
	syncManager := sync.NewSynchronizerManager(c)
	holder := jwe.NewRSAKeyHolder(syncManager.Secret("", ""))
	return jwe.NewJWETokenManager(holder)
}

func TestCreateHTTPAPIHandler(t *testing.T) {
	cManager := client.NewClientManager("", "http://localhost:8080")
	authManager := auth.NewAuthManager(cManager, getTokenManager(), authApi.AuthenticationModes{}, true)
	sManager := settings.NewSettingsManager()
	sbManager := systembanner.NewSystemBannerManager("Hello world!", "INFO")
	e := gin.Default()
	err := CreateHTTPAPIHandler(nil, cManager, authManager, sManager, sbManager, e)
	if err != nil {
		t.Fatal("CreateHTTPAPIHandler() cannot create HTTP API handler")
	}
}

func TestShouldDoCsrfValidation(t *testing.T) {
	c1, _ := gin.CreateTestContext(httptest.NewRecorder())
	c1.Request = &http.Request{
		Method: "PUT",
	}

	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = &http.Request{
		Method: "POST",
	}

	cases := []struct {
		request  *gin.Context
		expected bool
	}{
		{
			c1,
			false,
		},
		{
			c2,
			true,
		},
	}
	for _, c := range cases {
		actual := shouldDoCsrfValidation(c.request)
		if actual != c.expected {
			t.Errorf("shouldDoCsrfValidation(%#v) returns %#v, expected %#v", c.request, actual, c.expected)
		}
	}
}

func TestMapUrlToResource(t *testing.T) {
	cases := []struct {
		url, expected string
	}{
		{
			"/api/v1/pod",
			"pod",
		},
		{
			"/api/v1/node",
			"node",
		},
	}
	for _, c := range cases {
		actual := mapUrlToResource(c.url)
		if !reflect.DeepEqual(actual, &c.expected) {
			t.Errorf("mapUrlToResource(%#v) returns %#v, expected %#v", c.url, actual, c.expected)
		}
	}
}

func TestFormatRequestLog(t *testing.T) {
	cases := []struct {
		method      string
		uri         string
		content     map[string]string
		expected    string
		apiLogLevel string
	}{
		{
			"PUT",
			"/api/v1/pod",
			map[string]string{},
			"Incoming HTTP/1.1 PUT /api/v1/pod request",
			"DEFAULT",
		},
		{
			"PUT",
			"/api/v1/pod",
			map[string]string{},
			"",
			"NONE",
		},
		{
			"POST",
			"/api/v1/login",
			map[string]string{"password": "abc123"},
			"Incoming HTTP/1.1 POST /api/v1/login request from : { contents hidden }",
			"DEFAULT",
		},
		{
			"POST",
			"/api/v1/login",
			map[string]string{},
			"",
			"NONE",
		},
		{
			"POST",
			"/api/v1/login",
			map[string]string{"password": "abc123"},
			"Incoming HTTP/1.1 POST /api/v1/login request from : {\"password\":\"abc123\"}",
			"DEBUG",
		},
	}

	for _, c := range cases {
		jsonValue, _ := json.Marshal(c.content)

		req, err := http.NewRequest(c.method, c.uri, bytes.NewReader(jsonValue))
		req.Header.Set("Content-Type", "application/json")

		if err != nil {
			t.Error("Cannot mockup request")
		}

		builder := args.GetHolderBuilder()
		builder.SetAPILogLevel(c.apiLogLevel)
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = req

		actual := formatRequestLog(c)
		if !strings.Contains(actual, c.Request.RequestURI) {
			t.Errorf("formatRequestLog(%#v) returns %#v, expected to contain %#v", req, actual, c.Request.RequestURI)
		}
	}
}
