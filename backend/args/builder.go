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

import "net"

var builder = &holderBuilder{holder: Holder}

// Used to build argument holder structure. It is private to make sure that only 1 instance can be created
// that modifies singleton instance of argument holder.
type holderBuilder struct {
	holder *holder
}

// SetInsecurePort 'insecure-port' argument of Dashboard binary.
func (b *holderBuilder) SetInsecurePort(port int) *holderBuilder {
	b.holder.insecurePort = port
	return b
}

// SetPort 'port' argument of Dashboard binary.
func (b *holderBuilder) SetPort(port int) *holderBuilder {
	b.holder.port = port
	return b
}

// SetTokenTTL 'token-ttl' argument of Dashboard binary.
func (b *holderBuilder) SetTokenTTL(ttl int) *holderBuilder {
	b.holder.tokenTTL = ttl
	return b
}

// SetMetricClientCheckPeriod 'metric-client-check-period' argument of Dashboard binary.
func (b *holderBuilder) SetMetricClientCheckPeriod(period int) *holderBuilder {
	b.holder.metricClientCheckPeriod = period
	return b
}

// SetInsecureBindAddress 'insecure-bind-address' argument of Dashboard binary.
func (b *holderBuilder) SetInsecureBindAddress(ip net.IP) *holderBuilder {
	b.holder.insecureBindAddress = ip
	return b
}

// SetBindAddress 'bind-address' argument of Dashboard binary.
func (b *holderBuilder) SetBindAddress(ip net.IP) *holderBuilder {
	b.holder.bindAddress = ip
	return b
}

// SetDefaultCertDir 'default-cert-dir' argument of Dashboard binary.
func (b *holderBuilder) SetDefaultCertDir(certDir string) *holderBuilder {
	b.holder.defaultCertDir = certDir
	return b
}

// SetCertFile 'tls-cert-file' argument of Dashboard binary.
func (b *holderBuilder) SetCertFile(certFile string) *holderBuilder {
	b.holder.certFile = certFile
	return b
}

// SetKeyFile 'tls-key-file' argument of Dashboard binary.
func (b *holderBuilder) SetKeyFile(keyFile string) *holderBuilder {
	b.holder.keyFile = keyFile
	return b
}

// SetApiServerHost 'api-server-host' argument of Dashboard binary.
func (b *holderBuilder) SetApiServerHost(apiServerHost string) *holderBuilder {
	b.holder.apiServerHost = apiServerHost
	return b
}

// SetMetricsProvider 'metrics-provider' argument of Dashboard binary.
func (b *holderBuilder) SetMetricsProvider(metricsProvider string) *holderBuilder {
	b.holder.metricsProvider = metricsProvider
	return b
}

// SetHeapsterHost 'heapster-host' argument of Dashboard binary.
func (b *holderBuilder) SetHeapsterHost(heapsterHost string) *holderBuilder {
	b.holder.heapsterHost = heapsterHost
	return b
}

// SetSidecarHost 'sidecar-host' argument of Dashboard binary.
func (b *holderBuilder) SetSidecarHost(sidecarHost string) *holderBuilder {
	b.holder.sidecarHost = sidecarHost
	return b
}

// SetKubeConfigFile 'kubeconfig' argument of Dashboard binary.
func (b *holderBuilder) SetKubeConfigFile(kubeConfigFile string) *holderBuilder {
	b.holder.kubeConfigFile = kubeConfigFile
	return b
}

// SetSystemBanner 'system-banner' argument of Dashboard binary.
func (b *holderBuilder) SetSystemBanner(systemBanner string) *holderBuilder {
	b.holder.systemBanner = systemBanner
	return b
}

// SetSystemBannerSeverity 'system-banner-severity' argument of Dashboard binary.
func (b *holderBuilder) SetSystemBannerSeverity(systemBannerSeverity string) *holderBuilder {
	b.holder.systemBannerSeverity = systemBannerSeverity
	return b
}

// SetLogLevel 'api-log-level' argument of Dashboard binary.
func (b *holderBuilder) SetAPILogLevel(apiLogLevel string) *holderBuilder {
	b.holder.apiLogLevel = apiLogLevel
	return b
}

// SetAuthenticationMode 'authentication-mode' argument of Dashboard binary.
func (b *holderBuilder) SetAuthenticationMode(authMode []string) *holderBuilder {
	b.holder.authenticationMode = authMode
	return b
}

// SetAutoGenerateCertificates 'auto-generate-certificates' argument of Dashboard binary.
func (b *holderBuilder) SetAutoGenerateCertificates(autoGenerateCertificates bool) *holderBuilder {
	b.holder.autoGenerateCertificates = autoGenerateCertificates
	return b
}

// SetEnableInsecureLogin 'enable-insecure-login' argument of Dashboard binary.
func (b *holderBuilder) SetEnableInsecureLogin(enableInsecureLogin bool) *holderBuilder {
	b.holder.enableInsecureLogin = enableInsecureLogin
	return b
}

// SetDisableSettingsAuthorizer 'disable-settings-authorizer' argument of Dashboard binary.
func (b *holderBuilder) SetDisableSettingsAuthorizer(disableSettingsAuthorizer bool) *holderBuilder {
	b.holder.disableSettingsAuthorizer = disableSettingsAuthorizer
	return b
}

// SetEnableSkipLogin 'enable-skip-login' argument of Dashboard binary.
func (b *holderBuilder) SetEnableSkipLogin(enableSkipLogin bool) *holderBuilder {
	b.holder.enableSkipLogin = enableSkipLogin
	return b
}

// SetNamespace 'namespace' argument of Dashboard binary.
func (b *holderBuilder) SetNamespace(namespace string) *holderBuilder {
	b.holder.namespace = namespace
	return b
}

// SetLocaleConfig 'locale-config' argument of Dashboard binary.
func (b *holderBuilder) SetLocaleConfig(localeConfig string) *holderBuilder {
	b.holder.localeConfig = localeConfig
	return b
}

// GetHolderBuilder returns singleton instance of argument holder builder.
func GetHolderBuilder() *holderBuilder {
	return builder
}
