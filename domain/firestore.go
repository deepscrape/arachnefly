package domain

import "cloud.google.com/go/firestore"

type FirebaseManagerOpts struct {
	CredsFile    string
	DatabaseName string
	ProjectID    string
}

type IFirestore interface {
	CreateMachine(userID string, machine map[string]interface{}, deploymentId string, isDefault bool) error
	CreateMetrics(metrics *DeploymentMetrics) error

	SaveDeployment(userID string, deployment *MachineConfig) (string, error)

	UpdateMachine(userID string, machineID string, updates []firestore.Update) error
	DeleteMachine(userID string, machineID string) error
}
