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

package sync

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"

	"github.com/ycyxuehan/dashboard-gin/backend/errors"
	syncApi "github.com/ycyxuehan/dashboard-gin/backend/sync/api"
	"github.com/ycyxuehan/dashboard-gin/backend/sync/poll"
)

// Time interval between which secret should be resynchronized.
const secretSyncPeriod = 5 * time.Minute

// Implements Synchronizer interface. See Synchronizer for more information.
type secretSynchronizer struct {
	namespace string
	name      string

	secret         *v1.Secret
	client         kubernetes.Interface
	actionHandlers map[watch.EventType][]syncApi.ActionHandlerFunction
	errChan        chan error
	poller         syncApi.Poller

	mux sync.Mutex
}

// Name implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) Name() string {
	return fmt.Sprintf("%s-%s", s.name, s.namespace)
}

// Start implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) Start() {
	s.errChan = make(chan error)
	watcher, err := s.watch(s.namespace, s.name)
	if err != nil {
		s.errChan <- err
		close(s.errChan)
		return
	}

	go func() {
		log.Printf("Starting secret synchronizer for %s in namespace %s", s.name, s.namespace)
		defer watcher.Stop()
		defer close(s.errChan)
		for {
			select {
			case ev, ok := <-watcher.ResultChan():
				if !ok {
					s.errChan <- fmt.Errorf("%s watch ended with timeout", s.Name())
					return
				}
				if err := s.handleEvent(ev); err != nil {
					s.errChan <- err
					return
				}
			}
		}
	}()
}

// Error implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) Error() chan error {
	return s.errChan
}

// Create implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) Create(obj runtime.Object) error {
	secret := s.getSecret(obj)
	_, err := s.client.CoreV1().Secrets(secret.Namespace).Create( context.TODO(),secret, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// Get implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) Get() runtime.Object {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.secret == nil {
		// In case secret was not yet initialized try to do it synchronously
		secret, err := s.client.CoreV1().Secrets(s.namespace).Get( context.TODO(), s.name, metaV1.GetOptions{})
		if err != nil {
			return nil
		}

		log.Printf("Initializing secret synchronizer synchronously using secret %s from namespace %s", s.name,
			s.namespace)
		s.secret = secret
	}

	return s.secret
}

// Update implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) Update(obj runtime.Object) error {
	secret := s.getSecret(obj)
	_, err := s.client.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), secret, metaV1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// Delete implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) Delete() error {
	return s.client.CoreV1().Secrets(s.namespace).Delete(context.TODO(), s.name, metaV1.DeleteOptions{GracePeriodSeconds: new(int64)})
}

// RegisterActionHandler implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) RegisterActionHandler(handler syncApi.ActionHandlerFunction, events ...watch.EventType) {
	for _, ev := range events {
		if _, exists := s.actionHandlers[ev]; !exists {
			s.actionHandlers[ev] = make([]syncApi.ActionHandlerFunction, 0)
		}

		s.actionHandlers[ev] = append(s.actionHandlers[ev], handler)
	}
}

// Refresh implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) Refresh() {
	s.mux.Lock()
	defer s.mux.Unlock()

	secret, err := s.client.CoreV1().Secrets(s.namespace).Get( context.TODO(),s.name, metaV1.GetOptions{})
	if err != nil {
		log.Printf("Secret synchronizer %s failed to refresh secret", s.Name())
		return
	}

	s.secret = secret
}

// SetPoller implements Synchronizer interface. See Synchronizer for more information.
func (s *secretSynchronizer) SetPoller(poller syncApi.Poller) {
	s.poller = poller
}

func (s *secretSynchronizer) getSecret(obj runtime.Object) *v1.Secret {
	secret, ok := obj.(*v1.Secret)
	if !ok {
		panic("Provided object has to be a secret. Most likely this is a programming error")
	}

	return secret
}

func (s *secretSynchronizer) watch(namespace, name string) (watch.Interface, error) {
	if s.poller == nil {
		s.poller = poll.NewSecretPoller(name, namespace, s.client)
	}

	return s.poller.Poll(secretSyncPeriod), nil
}

func (s *secretSynchronizer) handleEvent(event watch.Event) error {
	for _, handler := range s.actionHandlers[event.Type] {
		handler(event.Object)
	}

	switch event.Type {
	case watch.Added:
		secret, ok := event.Object.(*v1.Secret)
		if !ok {
			return errors.NewInternal(fmt.Sprintf("Expected secret got %s", reflect.TypeOf(event.Object)))
		}

		s.update(*secret)
	case watch.Modified:
		secret, ok := event.Object.(*v1.Secret)
		if !ok {
			return errors.NewInternal(fmt.Sprintf("Expected secret got %s", reflect.TypeOf(event.Object)))
		}

		s.update(*secret)
	case watch.Deleted:
		s.mux.Lock()
		s.secret = nil
		s.mux.Unlock()
	case watch.Error:
		return errors.NewUnexpectedObject(event.Object)
	}

	return nil
}

func (s *secretSynchronizer) update(secret v1.Secret) {
	if reflect.DeepEqual(s.secret, &secret) {
		// Skip update if existing object is the same as new one
		return
	}

	s.mux.Lock()
	s.secret = &secret
	s.mux.Unlock()
}
