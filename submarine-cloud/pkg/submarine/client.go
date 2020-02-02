/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package submarine

import (
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/golang/glog"
	"net/http"
	"time"
)

// ClientInterface submarine client interface
type ClientInterface interface {
	// Close closes the connection.
	Close() error

	// Cmd calls the given Submarine command.
	///Cmd(cmd string, args ...interface{}) *submarine.Resp

	// PipeAppend adds the given call to the pipeline queue.
	// Use PipeResp() to read the response.
	///PipeAppend(cmd string, args ...interface{})

	// PipeResp returns the reply for the next request in the pipeline queue. Err
	// with ErrPipelineEmpty is returned if the pipeline queue is empty.
	///PipeResp() *submarine.Resp

	// PipeClear clears the contents of the current pipeline queue, both commands
	// queued by PipeAppend which have yet to be sent and responses which have yet
	// to be retrieved through PipeResp. The first returned int will be the number
	// of pending commands dropped, the second will be the number of pending
	// responses dropped
	///PipeClear() (int, int)

	// ReadResp will read a Resp off of the connection without sending anything
	// first (useful after you've sent a SUSBSCRIBE command). This will block until
	// a reply is received or the timeout is reached (returning the IOErr). You can
	// use IsTimeout to check if the Resp is due to a Timeout
	//
	// Note: this is a more low-level function, you really shouldn't have to
	// actually use it unless you're writing your own pub/sub code
	///ReadResp() *submarine.Resp

	// GetClusterAddress calls the given Submarine cluster server address list
	GetClusterAddress() ([]string, error)

	// GetClusterNodes calls the given Submarine cluster node list
	GetClusterNodes() (string, error)
}

// Client structure representing a client connection to submarine
type Client struct {
	commandsMapping map[string]string
	client          *resty.Client
	//client ClientInterface
}

func (c *Client) Close() error {
	panic("implement me")
}

const getClusterAddressUrl = "/api/v1/cluster/address"
const ClusterNodesUrl = "/api/v1/cluster/nodes"

// NewClient build a client connection and connect to a submarine address
func NewClient(addr string, cnxTimeout time.Duration, commandsMapping map[string]string) (ClientInterface, error) {
	var err error
	c := &Client{
		commandsMapping: commandsMapping,
	}

	// Create a Resty Client
	c.client = resty.New()

	// Retries are configured per client
	c.client.SetHostURL("http://"+addr).
		// Set retry count to non zero to enable retries
		SetRetryCount(3).
		// You can override initial retry wait time.
		// Default is 100 milliseconds.
		SetRetryWaitTime(3 * time.Second).
		// MaxWaitTime can be overridden as well.
		// Default is 2 seconds.
		SetRetryMaxWaitTime(10 * time.Second).
		// SetRetryAfter sets callback to calculate wait time between retries.
		// Default (nil) implies exponential backoff with jitter
		SetRetryAfter(func(client *resty.Client, resp *resty.Response) (time.Duration, error) {
			return 0, errors.New("quota exceeded")
		}).
		AddRetryCondition(
			// RetryConditionFunc type is for retry condition function
			// input: non-nil Response OR request execution error
			func(r *resty.Response, err error) bool {
				return r.StatusCode() == http.StatusTooManyRequests
			},
		)

	if err := c.dialTimeout(getClusterAddressUrl); err != nil {
		return nil, err
	}

	return c, err
}

func (c *Client) dialTimeout(url string) error {
	glog.Infof("dialTimeout(%s)", url)

	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		EnableTrace().
		Get(url)

	if err != nil || resp.StatusCode() != 200 {
		// Explore response object
		fmt.Println("Response Info:")
		fmt.Println("Error      :", err)
		fmt.Println("Status Code:", resp.StatusCode())
		fmt.Println("Status     :", resp.Status())
		fmt.Println("Time       :", resp.Time())
		fmt.Println("Received At:", resp.ReceivedAt())
		fmt.Println("Body       :\n", resp)
		fmt.Println()

		// Explore trace info
		fmt.Println("Request Trace Info:")
		ti := resp.Request.TraceInfo()
		fmt.Println("DNSLookup    :", ti.DNSLookup)
		fmt.Println("ConnTime     :", ti.ConnTime)
		fmt.Println("TLSHandshake :", ti.TLSHandshake)
		fmt.Println("ServerTime   :", ti.ServerTime)
		fmt.Println("ResponseTime :", ti.ResponseTime)
		fmt.Println("TotalTime    :", ti.TotalTime)
		fmt.Println("IsConnReused :", ti.IsConnReused)
		fmt.Println("IsConnWasIdle:", ti.IsConnWasIdle)
		fmt.Println("ConnIdleTime :", ti.ConnIdleTime)

		return err
	}
	return nil
}

// GetClusterAddress calls the given Submarine cluster server address list.
func (c *Client) GetClusterAddress() ([]string, error) {
	var clusterAddress []string
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		EnableTrace().
		Get(getClusterAddressUrl)
	if err != nil {
		return nil, err
	}

	glog.Infof("resp.Body() = %v", resp.Body())

	return clusterAddress, nil
}

// GetClusterAddress calls the given Submarine cluster server address list.
func (c *Client) GetClusterNodes() (string, error) {
	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		EnableTrace().
		Get(ClusterNodesUrl)
	if err != nil {
		return "", err
	}

	//glog.Infof("clusterAddress = %v", clusterAddress)
	glog.Infof("resp = %v", resp)
	glog.Infof("resp.Body() = %v", string(resp.Body()))
	glog.Infof("resp.String() = %v", resp.String())

	resp.RawBody()

	var strResp = `
	"{
		"status": "OK",
		"code": 200,
		"success": true,
		"message": null,
		"result": [{
			"NODE_NAME": "node-name1",
			"properties": {
				"STATUS": "OFFLINE",
				"INTP_PROCESS_LIST": ["SubmarineServerClusterTest"],
				"LATEST_HEARTBEAT": "2020-02-02T17:20:08.546",
				"SERVER_START_TIME": "2020-02-02T17:20:33.542",
				"MEMORY_USED / MEMORY_CAPACITY": "44.88GB / 64.00GB = 70.13%",
				"CPU_USED / CPU_CAPACITY": "0.00 / 12.00 = 0.00%"
			}
		},{
			"NODE_NAME": "node-name2",
			"properties": {
				"STATUS": "OFFLINE",
				"INTP_PROCESS_LIST": ["SubmarineServerClusterTest"],
				"LATEST_HEARTBEAT": "2020-02-02T17:20:08.546",
				"SERVER_START_TIME": "2020-02-02T17:20:33.542",
				"MEMORY_USED / MEMORY_CAPACITY": "44.88GB / 64.00GB = 70.13%",
				"CPU_USED / CPU_CAPACITY": "0.00 / 12.00 = 0.00%"
			}
		},{
			"NODE_NAME": "node-name3",
			"properties": {
				"STATUS": "OFFLINE",
				"INTP_PROCESS_LIST": ["SubmarineServerClusterTest"],
				"LATEST_HEARTBEAT": "2020-02-02T17:20:08.546",
				"SERVER_START_TIME": "2020-02-02T17:20:33.542",
				"MEMORY_USED / MEMORY_CAPACITY": "44.88GB / 64.00GB = 70.13%",
				"CPU_USED / CPU_CAPACITY": "0.00 / 12.00 = 0.00%"
			}
		}],
		"attributes": {}
	}`

	return strResp, nil
}
