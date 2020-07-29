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
	"github.com/ycyxuehan/dashboard-gin/backend/integration/api"
)

// IntegrationsGetter is responsible for listing all supported integrations.
type IntegrationsGetter interface {
	// List returns list of all supported integrations.
	List() []api.Integration
}

// List implements integration getter interface. See IntegrationsGetter for
// more information.
func (self *integrationManager) List() []api.Integration {
	result := make([]api.Integration, 0)

	// Append all types of integrations
	result = append(result, self.Metric().List()...)

	return result
}
