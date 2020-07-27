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
	"fmt"
	"log"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	syncApi "github.com/ycyxuehan/dashboard-gin/backend/sync/api"
)

// Overwatch is watching over every registered synchronizer. In case of error it will be logged and if RestartPolicy
// is set to "Always" synchronizer will be restarted.
var Overwatch *overwatch

// Initializes and starts Overwatch instance. It is private to make sure that only one instance is running.
func init() {
	Overwatch = &overwatch{
		syncMap:      make(map[string]syncApi.Synchronizer),
		policyMap:    make(map[string]RestartPolicy),
		restartCount: make(map[string]int),

		registrationSignal: make(chan string),
		restartSignal:      make(chan string),
	}

	log.Print("Starting overwatch")
	Overwatch.Run()
}

// RestartPolicy is used by Overwatch to determine how to behave in case of synchronizer error.
type RestartPolicy string

const (
	// In case of synchronizer error it will be restarted.
	AlwaysRestart RestartPolicy = "always"
	NeverRestart  RestartPolicy = "never"

	RestartDelay = 2 * time.Second
	// We don't need to sync it with every instance. If a single instance synchronizer fails
	// often, just force restart it.
	MaxRestartCount = 15
)

type overwatch struct {
	syncMap      map[string]syncApi.Synchronizer
	policyMap    map[string]RestartPolicy
	restartCount map[string]int

	registrationSignal chan string
	restartSignal      chan string
}

// RegisterSynchronizer registers given synchronizer with given restart policy.
func (o *overwatch) RegisterSynchronizer(synchronizer syncApi.Synchronizer, policy RestartPolicy) {
	if _, exists := o.syncMap[synchronizer.Name()]; exists {
		log.Printf("Synchronizer %s is already registered. Skipping", synchronizer.Name())
		return
	}

	o.syncMap[synchronizer.Name()] = synchronizer
	o.policyMap[synchronizer.Name()] = policy
	o.broadcastRegistrationEvent(synchronizer.Name())
}

// Run starts overwatch.
func (o *overwatch) Run() {
	o.monitorRegistrationEvents()
	o.monitorRestartEvents()
}

func (o *overwatch) monitorRestartEvents() {
	go wait.Forever(func() {
		select {
		case name := <-o.restartSignal:
			if o.restartCount[name] > MaxRestartCount {
				panic(fmt.Sprintf("synchronizer %s restart limit execeeded. Restarting pod.", name))
			}

			log.Printf("Restarting synchronizer: %s.", name)
			synchronizer := o.syncMap[name]
			synchronizer.Start()
			o.monitorSynchronizerStatus(synchronizer)
		}
	}, 0)
}

func (o *overwatch) monitorRegistrationEvents() {
	go wait.Forever(func() {
		select {
		case name := <-o.registrationSignal:
			synchronizer := o.syncMap[name]
			log.Printf("New synchronizer has been registered: %s. Starting", name)
			o.monitorSynchronizerStatus(synchronizer)
			synchronizer.Start()
		}
	}, 0)
}

func (o *overwatch) monitorSynchronizerStatus(synchronizer syncApi.Synchronizer) {
	stopCh := make(chan struct{})
	name := synchronizer.Name()
	go wait.Until(func() {
		select {
		case err := <-synchronizer.Error():
			log.Printf("Synchronizer %s exited with error: %s", name, err.Error())
			if o.policyMap[name] == AlwaysRestart {
				// Wait a sec before restarting synchronizer in case it exited with error.
				time.Sleep(RestartDelay)
				o.broadcastRestartEvent(name)
				o.restartCount[name]++
			}

			close(stopCh)
		}
	}, 0, stopCh)
}

func (o *overwatch) broadcastRegistrationEvent(name string) {
	o.registrationSignal <- name
}

func (o *overwatch) broadcastRestartEvent(name string) {
	o.restartSignal <- name
}
