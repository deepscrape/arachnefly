package database

import (
	"testing"

	"github.com/deepscrape/arachnefly/domain"
)

func TestCreateDocumentInNonExistentCollection(t *testing.T) {
	// Create a mock logger
	// logger, _ := zap.NewDevelopment()

	// Define FirestoreDBManagerOpts with valid credentials
	opts := &domain.FirebaseManagerOpts{
		ProjectID:    "libnet-d76db",
		DatabaseName: "easyscrape",
		CredsFile:    "../../libnet-d76db-949683c2222d.json",
	}

	// Call NewFirestoreClient to create a FirestoreDBManager instance
	firestoreManager, err := NewFirebaseClient(opts)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer firestoreManager.Close()

	// Attempt to create a document in a non-existent collection
	err = firestoreManager.CreateDocument("nonExistentCollection", "docID", map[string]interface{}{"field": "value"})

	// Assert that an error occurred
	if err == nil {
		t.Fatal("Expected an error when creating a document in a non-existent collection, got nil")
	}
}
