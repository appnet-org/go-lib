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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"

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
		loadMap: make(map[int](struct {
			load int
			ts   time.Time
		})),
	}
}

type appnetlbPicker struct {
	// subConns is the snapshot of the roundrobin balancer when this picker was
	// created. The slice is immutable. Each Get() will do a round robin
	// selection from it and return the selected SubConn.
	subConns   []balancer.SubConn
	next       uint32
	sharedData *sync.Map
	loadMap    map[int](struct {
		load int
		ts   time.Time
	})
}

// randomSelect randomly selects n elements from the list
func randomSelect(list []int, n int) []int {
	if n >= len(list) {
		return list
	}

	selected := make([]int, 0, n)
	perm := rand.Perm(len(list)) // Generate a random permutation of indices

	for i := 0; i < n; i++ {
		selected = append(selected, list[perm[i]])
	}

	return selected
}

func timeDiff(current, last time.Time) time.Duration {
	return current.Sub(last)
}

func (p *appnetlbPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	md, _ := metadata.FromOutgoingContext(info.Ctx)

	logger.Warningf("testlbPicker: picker called with md: %v", md)
	logger.Warningf("testlbPicker: picker called with shard-key: %v", md["shard-key"])

	url := fmt.Sprintf("http://10.96.88.99:8080/getReplica?key=%v&service=ServiceB", md["shard-key"][0])
	resp, err := http.Get(url)
	if err != nil {
		// handle the error
		logger.Warningf("Error getting response: %v", err)
		return balancer.PickResult{}, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	// Define a structure to hold the JSON response
	var result struct {
		ReplicaID []int `json:"replica_id"`
	}

	// Parse the JSON response
	if err := json.Unmarshal(body, &result); err != nil {
		panic("Error parsing JSON")
	}

	// Now you have a list of ints in result.ReplicaID
	logger.Warningf("Parsed Replica IDs: %v", result.ReplicaID)

	rand.Seed(42)
	selected := randomSelect(result.ReplicaID, 1)
	logger.Warningf("Randomly selected Replica IDs: %v", selected)

	// Apply the logic to the selected backends
	currentTime := time.Now()
	epsilon := 10 * time.Second
	for _, backend := range selected {
		resultTuple, exist := p.loadMap[backend]
		needToProbe := false
		if !exist {
			needToProbe = true
		} else {
			_, lastTs := resultTuple.load, resultTuple.ts
			freshness := timeDiff(currentTime, lastTs) - epsilon
			if freshness <= 0 {
				needToProbe = true
			}
		}

		if needToProbe {
			// backendLoad, lastTs = get(loadMapGlobal, backend)
			// set(loadMap, backend, backendLoad, lastTs)

			// How to get backendLoad and lastTs?
			// curl "http://10.96.88.97/getLoadInfo?service-name=my-service&replica-ids=0,1,2"
			// {"0":{"load":7,"timestamp":1724368802.8939054},"1":{"load":5,"timestamp":1724368802.8939054},"2":{"load":5,"timestamp":1724368802.8939054}}

			url := fmt.Sprintf("http://10.96.88.97/getLoadInfo?service-name=my-service&replica-ids=%v", backend)
			resp, err := http.Get(url)
			if err != nil {
				logger.Warningf("Error getting load info: %v", err)
				continue
			}
			defer resp.Body.Close()

			body, _ := ioutil.ReadAll(resp.Body)

			// Define a structure to hold the JSON response
			var result struct {
				Load      int     `json:"load"`
				Timestamp float64 `json:"timestamp"`
			}

			// Parse the JSON response
			if err := json.Unmarshal(body, &result); err != nil {
				// print the response as a string
				logger.Warningf("Error parsing JSON: %v", string(body))
				panic("Error parsing JSON")
			}

			backendLoad, lastTs := result.Load, time.Unix(int64(result.Timestamp), 0)
			p.loadMap[backend] = struct {
				load int
				ts   time.Time
			}{backendLoad, lastTs}
		}
	}

	selectedServer := 0
	minLoad := 1000000000

	for replicaId := range result.ReplicaID {
		// if p.loadMap has the load info for replicaId
		if _, exist := p.loadMap[replicaId]; exist {
			if p.loadMap[replicaId].load < minLoad {
				selectedServer = replicaId
				minLoad = p.loadMap[replicaId].load
			}
		}
	}

	// subConnsLen := uint32(len(p.subConns))
	// nextIndex := atomic.AddUint32(&p.next, 1)

	sc := p.subConns[selectedServer]
	return balancer.PickResult{SubConn: sc}, nil
}
