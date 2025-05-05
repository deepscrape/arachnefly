package routine

import (
	"time"

	"github.com/deepscrape/arachnefly/domain"
)

type IGlobal interface {
	GetCurrentTime() time.Time
	GetMachineDetails(machineId, flyApiUrl, flyApp string) (map[string]interface{}, error)
	FlyRequest(method string, url string, body interface{}, headers map[string]string) (map[string]interface{}, error)
	BuildConfigMap(machineConfig domain.MachineConfig) map[string]interface{}
	BuildEnvVars(environmentVariables []map[string]interface{}) map[string]interface{}
	GetImageSource(imageOption string, machineConfig domain.MachineConfig) (string, error)
	RemoveURLScheme(input string) string
}
