/*
 *
 * Copyright 2017 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
)

// Name is the name of appnet_lb balancer.
const Name = "appnet_lb"

var logger = grpclog.Component("appnetlb")

// newBuilder creates a new appnet balancer builder.
func NewBuilder(sharedData *sync.Map) balancer.Builder {
	return base.NewBalancerBuilder(Name, &appnetlbPickerBuilder{sharedData: sharedData}, base.Config{HealthCheck: true})
}

type appnetlbPickerBuilder struct {
	sharedData *sync.Map
}

func (b *appnetlbPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	logger.Warningf("testlbPicker: Build called with info: %v", info)
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	scs := make([]balancer.SubConn, 0, len(info.ReadySCs))
	for sc := range info.ReadySCs {
		scs = append(scs, sc)
	}

	return &appnetlbPicker{
		subConns: scs,
		// Start at a random index, as the same appnet balancer rebuilds a new
		// picker when SubConn states change, and we don't want to apply excess
		// load to the first server in the list.
		next:       uint32(rand.Intn(len(scs))),
		sharedData: b.sharedData,
	}
}

type appnetlbPicker struct {
	// subConns is the snapshot of the roundrobin balancer when this picker was
	// created. The slice is immutable. Each Get() will do a round robin
	// selection from it and return the selected SubConn.
	subConns   []balancer.SubConn
	next       uint32
	sharedData *sync.Map
}

func (p *appnetlbPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	md, _ := metadata.FromOutgoingContext(info.Ctx)

	logger.Warningf("testlbPicker: picker called with md: %v", md)

	resp, _ := http.Get("http://127.0.0.1:7379/PING")
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	logger.Warningf("testlbPicker: picker called with body: %v", string(body))

	// Example usage: getting a value from shared data
	if value, exists := p.sharedData.Load("exampleKey"); exists {
		logger.Warningf("testlbPicker: shared data value: %v", value)
	} else {
		logger.Warningf("roundrobinlbPicker: exampleKey not found")
	}

	// Example usage: getting a value from shared data
	if value, exists := p.sharedData.Load("testKey"); exists {
		logger.Warningf("testlbPicker: shared data value: %v", value)
	} else {
		logger.Warningf("testlbPicker: testKey1 not found")
	}

	subConnsLen := uint32(len(p.subConns))
	nextIndex := atomic.AddUint32(&p.next, 1)

	sc := p.subConns[nextIndex%subConnsLen]
	return balancer.PickResult{SubConn: sc}, nil
}
