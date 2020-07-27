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

package secret

import (
	"reflect"
	"testing"

	"github.com/ycyxuehan/dashboard-gin/backend/api"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/common"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/dataselect"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToSecretList(t *testing.T) {
	cases := []struct {
		secrets   []v1.Secret
		expected  *SecretList
		namespace *common.NamespaceQuery
	}{
		{
			[]v1.Secret{
				{
					ObjectMeta: metaV1.ObjectMeta{
						Name:              "user1",
						Namespace:         "foo",
						CreationTimestamp: metaV1.Unix(111, 222),
					},
				},
				{
					ObjectMeta: metaV1.ObjectMeta{
						Name:              "user2",
						Namespace:         "foo",
						CreationTimestamp: metaV1.Unix(111, 222),
					},
				},
			},
			&SecretList{
				Secrets: []Secret{
					{
						ObjectMeta: api.ObjectMeta{
							Name:              "user1",
							Namespace:         "foo",
							CreationTimestamp: metaV1.Unix(111, 222),
						},
						TypeMeta: api.NewTypeMeta(api.ResourceKindSecret),
					},
					{
						ObjectMeta: api.ObjectMeta{
							Name:              "user2",
							Namespace:         "foo",
							CreationTimestamp: metaV1.Unix(111, 222),
						},
						TypeMeta: api.NewTypeMeta(api.ResourceKindSecret),
					},
				},
				ListMeta: api.ListMeta{
					TotalItems: 2,
				},
			},
			common.NewNamespaceQuery([]string{"foo"}),
		},
	}

	for _, c := range cases {
		actual := ToSecretList(c.secrets, nil, dataselect.NoDataSelect)
		if !reflect.DeepEqual(actual, c.expected) {
			t.Errorf("toSecretList() ==\n%#v\nExpected: %#v", actual, c.expected)
		}
	}
}