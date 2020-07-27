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

package args

import (
	"net"

	"github.com/ycyxuehan/dashboard-gin/backend/cert/api"
)

//Holder is the object of holder
var Holder = &holder{}

// Argument holder structure. It is private to make sure that only 1 instance can be created. It holds all
// arguments values passed to Dashboard binary.
type holder struct {
	insecurePort            int
	port                    int
	tokenTTL                int
	metricClientCheckPeriod int

	insecureBindAddress net.IP
	bindAddress         net.IP

	defaultCertDir       string
	certFile             string
	keyFile              string
	apiServerHost        string
	metricsProvider      string
	heapsterHost         string
	sidecarHost          string
	kubeConfigFile       string
	systemBanner         string
	systemBannerSeverity string
	apiLogLevel          string
	namespace            string

	authenticationMode []string

	autoGenerateCertificates  bool
	enableInsecureLogin       bool
	disableSettingsAuthorizer bool

	enableSkipLogin bool

	localeConfig string
}

// GetInsecurePort 'insecure-port' argument of Dashboard binary.
func (h *holder) GetInsecurePort() int {
	return h.insecurePort
}

// GetPort 'port' argument of Dashboard binary.
func (h *holder) GetPort() int {
	return h.port
}

// GetTokenTTL 'token-ttl' argument of Dashboard binary.
func (h *holder) GetTokenTTL() int {
	return h.tokenTTL
}

// GetMetricClientCheckPeriod 'metric-client-check-period' argument of Dashboard binary.
func (h *holder) GetMetricClientCheckPeriod() int {
	return h.metricClientCheckPeriod
}

// GetInsecureBindAddress 'insecure-bind-address' argument of Dashboard binary.
func (h *holder) GetInsecureBindAddress() net.IP {
	return h.insecureBindAddress
}

// GetBindAddress 'bind-address' argument of Dashboard binary.
func (h *holder) GetBindAddress() net.IP {
	return h.bindAddress
}

// GetDefaultCertDir 'default-cert-dir' argument of Dashboard binary.
func (h *holder) GetDefaultCertDir() string {
	return h.defaultCertDir
}

// GetCertFile 'tls-cert-file' argument of Dashboard binary.
func (h *holder) GetCertFile() string {
	if len(h.certFile) == 0 && h.autoGenerateCertificates {
		return api.DashboardCertName
	}

	return h.certFile
}

// GetKeyFile 'tls-key-file' argument of Dashboard binary.
func (h *holder) GetKeyFile() string {
	if len(h.keyFile) == 0 && h.autoGenerateCertificates {
		return api.DashboardKeyName
	}

	return h.keyFile
}

// GetApiServerHost 'apiserver-host' argument of Dashboard binary.
func (h *holder) GetApiServerHost() string {
	return h.apiServerHost
}

// GetMetricsProvider 'metrics-provider' argument of Dashboard binary.
func (h *holder) GetMetricsProvider() string {
	return h.metricsProvider
}

// GetHeapsterHost 'heapster-host' argument of Dashboard binary.
func (h *holder) GetHeapsterHost() string {
	return h.heapsterHost
}

// GetSidecarHost 'sidecar-host' argument of Dashboard binary.
func (h *holder) GetSidecarHost() string {
	return h.sidecarHost
}

// GetKubeConfigFile 'kubeconfig' argument of Dashboard binary.
func (h *holder) GetKubeConfigFile() string {
	return h.kubeConfigFile
}

// GetSystemBanner 'system-banner' argument of Dashboard binary.
func (h *holder) GetSystemBanner() string {
	return h.systemBanner
}

// GetSystemBannerSeverity 'system-banner-severity' argument of Dashboard binary.
func (h *holder) GetSystemBannerSeverity() string {
	return h.systemBannerSeverity
}

// LogLevel 'api-log-level' argument of Dashboard binary.
func (h *holder) GetAPILogLevel() string {
	return h.apiLogLevel
}

// GetAuthenticationMode 'authentication-mode' argument of Dashboard binary.
func (h *holder) GetAuthenticationMode() []string {
	return h.authenticationMode
}

// GetAutoGenerateCertificates 'auto-generate-certificates' argument of Dashboard binary.
func (h *holder) GetAutoGenerateCertificates() bool {
	return h.autoGenerateCertificates
}

// GetEnableInsecureLogin 'enable-insecure-login' argument of Dashboard binary.
func (h *holder) GetEnableInsecureLogin() bool {
	return h.enableInsecureLogin
}

// GetDisableSettingsAuthorizer 'disable-settings-authorizer' argument of Dashboard binary.
func (h *holder) GetDisableSettingsAuthorizer() bool {
	return h.disableSettingsAuthorizer
}

// GetEnableSkipLogin 'enable-skip-login' argument of Dashboard binary.
func (h *holder) GetEnableSkipLogin() bool {
	return h.enableSkipLogin
}

// GetNamespace 'namespace' argument of Dashboard binary.
func (h *holder) GetNamespace() string {
	return h.namespace
}

// GetLocaleConfig 'locale-config' argument of Dashboard binary.
func (h *holder) GetLocaleConfig() string {
	return h.localeConfig
}
