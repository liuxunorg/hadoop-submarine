package submarine

type ClusterNodeResponse struct {
	Result []ClusterNode `json:"result"`
}

type ClusterNode struct {
	NodeName  string `json:"NODE_NAME"`
	Properties Properties `json:"properties"`
}

type Properties struct {
	Status          string   `json:"STATUS,omitempty"`
	IntpProcessList []string `json:"INTP_PROCESS_LIST,omitempty"`
	LatestHeartbeat string   `json:"LATEST_HEARTBEAT,omitempty"`
	ServerStartTime string   `json:"SERVER_START_TIME,omitempty"`
	MemoryUseRate   string   `json:"MEMORY_USED / MEMORY_CAPACITY,omitempty"`
	CpuUseRate      string   `json:"CPU_USED / CPU_CAPACITY,omitempty"`
}
