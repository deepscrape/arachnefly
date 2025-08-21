package domain

import "time"

// Define a struct to hold the incoming data
type MachineConfig struct {
	Default              bool      `json:"default"`
	ImageOption          string    `json:"imageOption"`
	DefaultImage         string    `json:"defaultImage"`
	CloneMachine         string    `json:"cloneMachine"`
	Dockerfile           string    `json:"dockerfile"`
	DockerHubUrl         string    `json:"dockerHubUrl"`
	MachineName          string    `json:"machineName"`
	Region               string    `json:"region"`
	CpuCores             int       `json:"cpuCores"`
	CpuType              string    `json:"cpuType"`
	Memory               int       `json:"memory"`
	AutoStart            bool      `json:"autoStart"`
	AutoStop             string    `json:"autoStop"`
	EnvironmentVariables string    `json:"environmentVariables"`
	FlyToml              string    `json:"flyToml"`
	CreatedAt            time.Time `json:"created_at,omitempty"` // Timestamp of creation
	UpdateAt             time.Time `json:"update_at,omitempty"`  // Timestamp of the last update
}

type MachineExecuteTask struct {
	Cmd       string   `json:"cmd"`                 // Deprecated: use Command instead
	Command   []string `json:"command,omitempty"`   // The command to execute
	Container string   `json:"container,omitempty"` // The container in which to execute the command
	Stdin     string   `json:"stdin,omitempty"`     // The stdin for the command
	Timeout   int      `json:"timeout,omitempty"`   // The timeout for the command in seconds
}

type EnvVars struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type DeploymentMetrics struct {
	UserID              string  `json:"user_id"`
	MachineID           string  `json:"machine_id"`
	CPUUsage            float64 `json:"cpu_usage"`
	MemoryUsage         float64 `json:"memory_usage"`
	NetworkTraffic      float64 `json:"network_traffic"`
	DiskIO              float64 `json:"disk_io"`
	ResponseTime        float64 `json:"response_time"`
	ErrorRate           float64 `json:"error_rate"`
	ExceptionRate       float64 `json:"exception_rate"`
	CrashRate           float64 `json:"crash_rate"`
	DiskSpaceUsed       float64 `json:"disk_space_used"`
	ActiveInstances     int     `json:"active_instances"`
	DeploymentFrequency float64 `json:"deployment_frequency"` // e.g., deployments per day
	DeploymentTime      float64 `json:"deployment_time"`      // in seconds
	RollbackRate        float64 `json:"rollback_rate"`
	Timestamp           int64   `json:"timestamp"`
}

type WaitForState struct {
	State      string `json:"state"`
	InstanceId string `json:"instance_id"`
	Timeout    int    `json:"timeout"`
}
