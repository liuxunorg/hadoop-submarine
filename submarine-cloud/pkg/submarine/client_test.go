package submarine

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGetClusterNodes(t *testing.T) {
	ts := createGetServer(t)
	defer ts.Close()

	resp, _ := dc().R().Get(ts.URL + ClusterNodesUrl)

	assertEqual(t, http.StatusOK, resp.StatusCode())
	t.Logf("resp.StatusCode() = %v", resp.StatusCode())
	t.Logf("resp.String() = %v", resp.String())

	var birds ClusterNodeResponse
	json.Unmarshal(resp.Body(), &birds)

	for _, res := range birds.Result {
		t.Logf("res.NodeName = %s", res.NodeName)
		t.Logf("res.Properties.CpuUseRate = %s", res.Properties.CpuUseRate)
		t.Logf("res.Properties.LatestHeartbeat = %s", res.Properties.LatestHeartbeat)
		t.Logf("res.Properties.MemoryUseRate = %s", res.Properties.MemoryUseRate)
		t.Logf("res.Properties.ServerStartTime = %s", res.Properties.ServerStartTime)
		t.Logf("res.Properties.Status = %s", res.Properties.Status)
		for _, intp := range res.Properties.IntpProcessList {
			t.Logf("res.Properties.IntpProcessList.intp = %s", intp)
		}
	}

	assertEqual(t, len(birds.Result), 3)
}

func createGetServer(t *testing.T) *httptest.Server {
	ts := createTestServer(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Method: %v", r.Method)
		t.Logf("Path: %v", r.URL.Path)

		if r.Method == resty.MethodGet {
			switch r.URL.Path {
			case ClusterNodesUrl:
				_, _ = w.Write([]byte(MockClusterNodesUrlRespBody))
			}
		}
	})

	return ts
}

func createTestServer(fn func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(fn))
}

func dc() *resty.Client {
	c := resty.New()
	return c
}

func assertEqual(t *testing.T, e, g interface{}) (r bool) {
	if !equal(e, g) {
		t.Errorf("Expected [%v], got [%v]", e, g)
	}

	return
}

func equal(expected, got interface{}) bool {
	return reflect.DeepEqual(expected, got)
}

var MockClusterNodesUrlRespBody = `
	{
		"status": "OK",
		"code": 200,
		"success": true,
		"message": null,
		"result": ` + ClusterNodesUrlResp + `,
		"attributes": {}
	}`

var ClusterNodesUrlResp = `
	[{
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
			"INTP_PROCESS_LIST": ["SubmarineServerClusterTest1", "SubmarineServerClusterTest2"],
			"LATEST_HEARTBEAT": "2020-02-02T17:20:08.546",
			"SERVER_START_TIME": "2020-02-02T17:20:33.542",
			"MEMORY_USED / MEMORY_CAPACITY": "44.88GB / 64.00GB = 70.13%",
			"CPU_USED / CPU_CAPACITY": "0.00 / 12.00 = 0.00%"
		}
	}]`
