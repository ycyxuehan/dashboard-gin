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

package cert

import (
	"crypto/tls"
	"log"
	"os"

	certapi "github.com/ycyxuehan/dashboard-gin/backend/cert/api"
)

// Manager is used to implement cert/api/types.Manager interface. See Manager for more information.
type Manager struct {
	creator certapi.Creator
	certDir string
}

// GetCertificates implements Manager interface. See Manager for more information.
func (m *Manager) GetCertificates() (tls.Certificate, error) {
	if m.keyFileExists() && m.certFileExists() {
		log.Println("Certificates already exist. Returning.")
		return tls.LoadX509KeyPair(
			m.path(m.creator.GetCertFileName()),
			m.path(m.creator.GetKeyFileName()),
		)
	}

	key := m.creator.GenerateKey()
	cert := m.creator.GenerateCertificate(key)
	log.Println("Successfully created certificates")
	keyPEM, certPEM, err := m.creator.KeyCertPEMBytes(key, cert)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(certPEM, keyPEM)
}

func (m *Manager) keyFileExists() bool {
	return m.exists(m.path(m.creator.GetKeyFileName()))
}

func (m *Manager) certFileExists() bool {
	return m.exists(m.path(m.creator.GetCertFileName()))
}

func (m *Manager) path(certFile string) string {
	return m.certDir + string(os.PathSeparator) + certFile
}

func (m *Manager) exists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

// NewCertManager creates Manager object.
func NewCertManager(creator certapi.Creator, certDir string) certapi.Manager {
	return &Manager{creator: creator, certDir: certDir}
}
