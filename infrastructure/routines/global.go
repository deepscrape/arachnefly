package routines

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/deepscrape/arachnefly/domain"
	"github.com/goccy/go-json"

	"github.com/valyala/fasthttp"
)

type Global struct {
	// encrypted *domain.EncryptedData
	// security *secrets.Security
	// logger   *zap.Logger
	flyApiToken string
}

func NewGlobalRoutines(flyApiToken string /* logger *zap.Logger */) *Global {
	return &Global{flyApiToken /* security: secrets.NewSecurity(logger), logger: logger */}
}

func (g *Global) GetCurrentTime() time.Time {
	return time.Now()
}

// 🚀 Get Machine IP
func (g *Global) GetMachineDetails(machineId, flyApiUrl, flyApp string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/apps/%s/machines/%s", flyApiUrl, flyApp, machineId)
	log.Println("Get Machine url:", url)
	return g.FlyRequest("GET", url, nil, nil)
}

// 🚀 Helper Function for Fly.io API Requests
func (g *Global) FlyRequest(method string, url string, body interface{}, headers map[string]string) (map[string]interface{}, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	res := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(res)

	req.Header.SetMethod(method)
	req.Header.Set("Authorization", "Bearer "+g.flyApiToken)
	req.Header.Set("Content-Type", "application/json")

	// Add optional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add URL to the request
	req.SetRequestURI(url)

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req.SetBody(jsonBody)
	}

	client := &fasthttp.Client{}
	err := client.Do(req, res)
	if err != nil {
		return nil, err
	}

	var responseData map[string]interface{}
	err = json.Unmarshal(res.Body(), &responseData)
	if err != nil {
		// Handle the error appropriately, e.g., log it or return an error response
		return nil, err
	}
	return responseData, nil
}

func (h *Global) BuildConfigMap(machineConfig domain.MachineConfig) map[string]interface{} {

	// Unmarshal the EnvironmentVariables JSON string into a slice of maps
	var envVars []map[string]interface{}
	err := json.Unmarshal([]byte(machineConfig.EnvironmentVariables), &envVars)
	if err != nil {
		log.Println("Error unmarshalling EnvironmentVariables:", err)
		return nil
	}

	// Build Environment Variables
	port := 8080
	env := h.BuildEnvVars(envVars)
	portStr, _ := env["PORT"].(string)

	port, _ = strconv.Atoi(portStr)
	log.Println("Port:", port)

	// Get the image source
	image, err := h.GetImageSource(machineConfig.ImageOption, machineConfig)
	if err != nil {
		log.Println("Error getting image source:", err)
		return nil
	}

	// Build the config map
	config := map[string]interface{}{
		"name":   machineConfig.MachineName, // Use the machine name from the request, should be a unique name
		"region": machineConfig.Region,
		"config": map[string]interface{}{
			"image":        image, // Use the image name from the request
			"auto_destroy": false, // Set auto_destroy based on AutoStop code
			"env":          env,
			"restart": map[string]interface{}{
				"policy": "always",
			},
			"guest": map[string]interface{}{
				// "size": machineConfig.CpuType,
				"cpu_kind":  machineConfig.CpuType,
				"cpus":      machineConfig.CpuCores,
				"memory_mb": machineConfig.Memory,
			},
			"services": []map[string]interface{}{
				{
					"internal_port": port,
					"protocol":      "tcp",
					"concurrency": map[string]interface{}{
						"type":       "connections",
						"soft_limit": 20,
						"hard_limit": 25,
					},
					"ports": []map[string]interface{}{
						{
							"port":     80,
							"handlers": []string{"http"},
						},
						{
							"port":     port,
							"handlers": []string{"http"},
						},
						{
							"port":     443,
							"handlers": []string{"tls", "http"},
						},
					},
					"autostop":             machineConfig.AutoStop,
					"autostart":            machineConfig.AutoStart,
					"min_machines_running": 0,
				},
				{
					"internal_port": 6379,
					"protocol":      "tcp",
					"concurrency": map[string]interface{}{
						"type":       "connections",
						"soft_limit": 20,
						"hard_limit": 25,
					},

					"ports": []map[string]interface{}{
						{
							"port": 6379,
						},
					},
				},
			},
		},
		"checks": map[string]interface{}{
			"httpget": map[string]interface{}{
				"type":     "http",
				"port":     port,
				"method":   "GET",
				"path":     "/",
				"interval": "15s",
				"timeout":  "10s",
			},
		},
	}
	return config
}

func (h *Global) BuildEnvVars(environmentVariables []map[string]interface{}) map[string]interface{} {
	fmt.Println("environmentVariables:", environmentVariables)
	env := make(map[string]interface{})
	for _, envVar := range environmentVariables {
		if key, ok := envVar["key"].(string); ok {
			if value, ok := envVar["value"].(string); ok {
				env[key] = value
			}
		}
	}
	return env
}

func (h *Global) GetImageSource(imageOption string, machineConfig domain.MachineConfig) (string, error) {
	switch imageOption {
	case "default":
		return machineConfig.DefaultImage, nil
	case "clone":
		return machineConfig.CloneMachine, nil
	case "url":
		return machineConfig.DockerHubUrl, nil
	case "upload":
		// Assuming the image is already uploaded and the URL is available in machineConfig.UploadURL
		// Or handle the upload logic here if it's not already handled.
		return machineConfig.Dockerfile, nil // Replace with actual upload dockerfile if different.
	default:
		return "", fmt.Errorf("invalid image option: %s", imageOption)
	}
}
