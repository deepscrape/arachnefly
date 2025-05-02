package domain

// Define a struct to hold the incoming data
type MachineConfig struct {
	ImageOption          string `json:"imageOption"`
	DefaultImage         string `json:"defaultImage"`
	CloneMachine         string `json:"cloneMachine"`
	Dockerfile           string `json:"dockerfile"`
	DockerHubUrl         string `json:"dockerHubUrl"`
	MachineName          string `json:"machineName"`
	Region               string `json:"region"`
	CpuCores             int    `json:"cpuCores"`
	CpuType              string `json:"cpuType"`
	Memory               int    `json:"memory"`
	AutoStart            bool   `json:"autoStart"`
	AutoStop             string `json:"autoStop"`
	EnvironmentVariables string `json:"environmentVariables"`
	FlyToml              string `json:"flyToml"`
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
