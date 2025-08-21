package repository

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/deepscrape/arachnefly/domain"
	"github.com/deepscrape/arachnefly/domain/routine"
	database "github.com/deepscrape/arachnefly/infrastructure/db"
	"github.com/deepscrape/arachnefly/infrastructure/routines"
	"github.com/xeipuuv/gojsonschema"
)

type Firestore struct {
	// Define The adapters
	db           *database.FirebaseManager
	schemaLoader gojsonschema.JSONLoader
	// cfg    *config.Config
	// logger *zap.Logger

	globalRoutines routine.IGlobal
	// security       secret.ISecurity
}

func NewFirestore(db *database.FirebaseManager, flyApiToken string /* , cfg *config.Config, logger *zap.Logger */) *Firestore {
	var schemaContent string
	// get the schema json from the schema file
	filename := "machine.schema.json"
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	filePath := filepath.Join(currentDir, filename)

	jsonFile, err := os.Open(filePath)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, _ := os.ReadFile(filePath)

	schemaContent = string(byteValue)

	// Load the schema
	schemaLoader := gojsonschema.NewStringLoader(schemaContent)

	return &Firestore{
		db:           db,
		schemaLoader: schemaLoader,
		// cfg:            cfg,
		// logger:         logger,
		globalRoutines: routines.NewGlobalRoutines(flyApiToken /* logger */),
		// security:       secrets.NewSecurity(logger),
	}
}

func (f *Firestore) CreateMachine(userID string, machine map[string]interface{}, deploymentId string, isDefault bool) error {
	// Get the user ID from the context
	// Create a new document in the "users" collection with the user's ID

	if machine["id"] == "" {
		return fmt.Errorf("machine ID is required")
	}

	// Validate the machine against the schema
	documentLoader := gojsonschema.NewGoLoader(machine)

	result, err := gojsonschema.Validate(f.schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	// Check if the validation was successful
	if !result.Valid() {
		log.Println("Schema validation failed. See errors:")
		for _, desc := range result.Errors() {
			log.Printf("- %s", desc)
		}
		return fmt.Errorf("schema validation failed")
	}

	// Create a new document in the "machine" subcollection with the user
	// set collection path
	collectionPath := fmt.Sprintf("users/%s/machines", userID)

	// map deploymentId and default variable to set default or not
	machine["deploymentId"] = deploymentId
	machine["default"] = isDefault
	err = f.db.CreateDocument(collectionPath, machine["id"].(string), machine)
	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	return nil
}

func (f *Firestore) UpdateMachine(userID string, machineID string, updates []firestore.Update) error {
	collectionName := fmt.Sprintf("users/%s/machines", userID)
	err := f.db.UpdateDocument(collectionName, machineID, updates)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	return nil
}

func (f *Firestore) DeleteMachine(userID string, machineID string) error {
	collectionName := fmt.Sprintf("users/%s/machines", userID)
	err := f.db.DeleteDocument(collectionName, machineID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

func (f *Firestore) CreateMetrics(metrics *domain.DeploymentMetrics) error {
	collectionName := "deploy_metrics"
	docID := metrics.UserID + "_" + metrics.MachineID + "_" + fmt.Sprintf("%d", metrics.Timestamp)

	err := f.db.CreateDocument(collectionName, docID, metrics)
	if err != nil {
		return fmt.Errorf("failed to create metrics document: %w", err)
	}

	return nil
}

func (f *Firestore) SaveDeployment(userID string, deployment *domain.MachineConfig) (string, error) {

	deployData := &domain.MachineConfig{
		Default:              deployment.Default,
		ImageOption:          deployment.ImageOption,
		DefaultImage:         deployment.DefaultImage,
		CloneMachine:         deployment.CloneMachine,
		Dockerfile:           deployment.Dockerfile,
		DockerHubUrl:         deployment.DockerHubUrl,
		MachineName:          deployment.MachineName,
		Region:               deployment.Region,
		CpuCores:             deployment.CpuCores,
		CpuType:              deployment.CpuType,
		Memory:               deployment.Memory,
		AutoStart:            deployment.AutoStart,
		AutoStop:             deployment.AutoStop,
		EnvironmentVariables: deployment.EnvironmentVariables,
		FlyToml:              deployment.FlyToml,
		CreatedAt:            time.Now(),
	}

	// Create a new document in the "users" collection with the user's ID
	collectionName := fmt.Sprintf("users/%s/deployments", userID)
	log.Printf("User ID: %s exists\n", userID)
	deployId, err := f.db.CreateDocumentID(collectionName, deployData)
	if err != nil {
		return "", fmt.Errorf("failed to create document: %w", err)
	}

	return deployId, nil
}
