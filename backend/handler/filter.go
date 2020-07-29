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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/xsrftoken"
	utilnet "k8s.io/apimachinery/pkg/util/net"

	"github.com/ycyxuehan/dashboard-gin/backend/args"
	authApi "github.com/ycyxuehan/dashboard-gin/backend/auth/api"
	clientapi "github.com/ycyxuehan/dashboard-gin/backend/client/api"
	"github.com/ycyxuehan/dashboard-gin/backend/errors"
	"github.com/ycyxuehan/dashboard-gin/backend/utils/httphelper"
)

const (
	originalForwardedForHeader = "X-Original-Forwarded-For"
	forwardedForHeader         = "X-Forwarded-For"
	realIPHeader               = "X-Real-Ip"
)

// InstallFilters installs defined filter for given web service
func InstallFilters(ws *gin.Engine, manager clientapi.ClientManager) {
	// ws.Filter(requestAndResponseLogger)
	// ws.Filter(metricsFilter)
	// ws.Filter(validateXSRFFilter(manager.CSRFKey()))
	// ws.Filter(restrictedResourcesFilter)
}

// Filter used to restrict access to dashboard exclusive resource, i.e. secret used to store dashboard encryption key.
func restrictedResourcesFilter(c *gin.Context, /*chain *restful.FilterChain*/) {
	if !authApi.ShouldRejectRequest(c.Request.URL.String()) {
		// chain.ProcessFilter(request, response)
		return
	}

	err := errors.NewUnauthorized(errors.MsgDashboardExclusiveResourceError)
	httphelper.RestfullResponse(c, int(err.ErrStatus.Code), err.Error())
}

// web-service filter function used for request and response logging.
func requestAndResponseLogger(c *gin.Context /*,	chain *restful.FilterChain*/) {
	if args.Holder.GetAPILogLevel() != "NONE" {
		log.Printf(formatRequestLog(c))
	}

	// chain.ProcessFilter(request, response)

	if args.Holder.GetAPILogLevel() != "NONE" {
		log.Printf(formatResponseLog(c))
	}
}

// formatRequestLog formats request log string.
func formatRequestLog(c *gin.Context) string {
	uri := ""
	content := "{}"

	if c.Request.URL != nil {
		uri = c.Request.URL.RequestURI()
	}

	byteArr, err := ioutil.ReadAll(c.Request.Body)
	if err == nil {
		content = string(byteArr)
	}

	// Restore request body so we can read it again in regular request handlers
	c.Request.Body = ioutil.NopCloser(bytes.NewReader(byteArr))

	// Is DEBUG level logging enabled? Yes?
	// Great now let's filter out any content from sensitive URLs
	if args.Holder.GetAPILogLevel() != "DEBUG" && checkSensitiveURL(&uri) {
		content = "{ contents hidden }"
	}

	return fmt.Sprintf(RequestLogString, time.Now().Format(time.RFC3339), c.Request.Proto,
		c.Request.Method, uri, getRemoteAddr(c.Request), content)
}

// formatResponseLog formats response log string.
func formatResponseLog(c *gin.Context) string {
	return fmt.Sprintf(ResponseLogString, time.Now().Format(time.RFC3339),
		getRemoteAddr(c.Request), c.Request.Response.Status)
}

// checkSensitiveUrl checks if a string matches against a sensitive URL
// true if sensitive. false if not.
func checkSensitiveURL(url *string) bool {
	var s struct{}
	var sensitiveUrls = make(map[string]struct{})
	sensitiveUrls["/api/v1/login"] = s
	sensitiveUrls["/api/v1/csrftoken/login"] = s
	sensitiveUrls["/api/v1/token/refresh"] = s

	if _, ok := sensitiveUrls[*url]; ok {
		return true
	}
	return false

}

func metricsFilter(c *gin.Context/*chain *restful.FilterChain*/) {
	resource := mapUrlToResource(c.Request.URL.Path)
	httpClient := utilnet.GetHTTPClient(c.Request)

	// chain.ProcessFilter(req, resp)

	if resource != nil {
		monitor(
			c.Request.Method,
			*resource, httpClient,
			c.GetHeader("Content-Type"),
			c.Request.Response.StatusCode,
			time.Now(),
		)
	}
}

func validateXSRFFilter(csrfKey string) gin.HandlerFunc {
	return func(c *gin.Context/*, chain *restful.FilterChain*/) {
		resource := mapUrlToResource(c.Request.URL.Path)

		if resource == nil || (shouldDoCsrfValidation(c) &&
			!xsrftoken.Valid(c.GetHeader("X-CSRF-TOKEN"), csrfKey, "none",
				*resource)) {
			err := errors.NewInvalid("CSRF validation failed")
			log.Print(err)
			c.Header("Content-Type", "text/plain")
			httphelper.RestfullResponse(c, http.StatusUnauthorized, err.Error()+"\n")
			return
		}

		// chain.ProcessFilter(c)
	}
}

// Post requests should set correct X-CSRF-TOKEN header, all other requests
// should either not edit anything or be already safe to CSRF attacks (PUT
// and DELETE)
func shouldDoCsrfValidation(c *gin.Context) bool {
	if c.Request.Method != http.MethodPost {
		return false
	}

	// Validation handlers are idempotent functions, and not actual data
	// modification operations
	if strings.HasPrefix(c.Request.URL.Path, "/api/v1/appdeployment/validate/") {
		return false
	}

	return true
}

// mapUrlToResource extracts the resource from the URL path /api/v1/<resource>.
// Ignores potential subresources.
func mapUrlToResource(url string) *string {
	parts := strings.Split(url, "/")
	if len(parts) < 3 {
		return nil
	}
	return &parts[3]
}

// getRemoteAddr extracts the remote address of the request, taking into
// account proxy headers.
func getRemoteAddr(r *http.Request) string {
	if ip := getRemoteIPFromForwardHeader(r, originalForwardedForHeader); ip != "" {
		return ip
	}

	if ip := getRemoteIPFromForwardHeader(r, forwardedForHeader); ip != "" {
		return ip
	}

	if realIP := strings.TrimSpace(r.Header.Get(realIPHeader)); realIP != "" {
		return realIP
	}

	return r.RemoteAddr
}

func getRemoteIPFromForwardHeader(r *http.Request, header string) string {
	ips := strings.Split(r.Header.Get(header), ",")
	return strings.TrimSpace(ips[0])
}
