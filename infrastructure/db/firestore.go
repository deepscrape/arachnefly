package database

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/deepscrape/arachnefly/domain"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Credentials struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

type FirebaseManager struct {
	ctx      context.Context
	keyspace string
	// logger   *zap.Loggervar
	firebaseAuth  *auth.Client
	firebaseStore *firestore.Client
	FirebaseApp   *firebase.App
}

func NewFirebaseClient(opts *domain.FirebaseManagerOpts /* , logger *zap.Logger */) (*FirebaseManager, error) {
	var firebaseAuth *auth.Client
	var FirebaseApp *firebase.App
	var options []option.ClientOption
	// var fireConfig *firebase.Config

	ctx := context.Background()

	if opts.ProjectID == "" {
		opts.ProjectID = "libnet-d76db" // default project ID
	}

	if opts.DatabaseName == "" {
		opts.DatabaseName = "easyscrape" // default database name
	}

	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost != "" {
		// Use emulator: no credentials needed
		log.Println("Using Firestore emulator at", emulatorHost)
	} else {

		if opts.CredsFile == "" {
			var err error
			opts.CredsFile, err = storeSecretFirebaseCredsAsFile()
			if err != nil {
				log.Fatalf("Error storing Firebase credentials: %v", err)
			} // default credentials file
		}

		// Override with -project flags
		// flag.StringVar(&opts.ProjectID, "libnet-d76db", opts.DatabaseName, "easyscrape")
		// flag.Parse()

		fmt.Println("Credentials file path:", opts.CredsFile)
		if _, err := os.Stat(opts.CredsFile); os.IsNotExist(err) {
			log.Fatalf("Credentials file does not exist: %s", opts.CredsFile)
		}

		options = append(options, option.WithCredentialsFile(opts.CredsFile))
	}

	client, err := firestore.NewClientWithDatabase(ctx, opts.ProjectID, opts.DatabaseName, options...)
	if err != nil {
		log.Fatal("Failed to create new Firestore client", zap.Error(err))
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("client is nil")
	}

	FirebaseApp, err = firebase.NewApp(ctx, &firebase.Config{ProjectID: opts.ProjectID}, options...)
	if err != nil {
		log.Fatalf("Firebase initialization error: %v", err)
	}

	firebaseAuth, err = FirebaseApp.Auth(ctx)
	if err != nil {
		log.Fatalf("Firebase Auth initialization error: %v", err)
	}

	/* docRef := client.Collection("nonExistentCollection").NewDoc()

	_, err = docRef.Create(ctx, map[string]interface{}{"field": "value"})
	if err != nil {
		log.Fatalf("Failed to create document: %v", err)
	} */

	log.Default().Println("Firestore client created successfully")
	return &FirebaseManager{
		ctx:      ctx,
		keyspace: "libnet-d76db",
		// logger:   logger,
		firebaseAuth:  firebaseAuth,
		firebaseStore: client,
		FirebaseApp:   FirebaseApp,
	}, nil
}

// create new colleciton firestore id
func (fs *FirebaseManager) CreateDocumentID(collection string, data interface{}) (string, error) {
	docRef := fs.firebaseStore.Collection(collection).NewDoc()
	_, err := docRef.Create(fs.ctx, data)
	if err != nil {
		return "", err
	}
	return docRef.ID, nil
}

func (fs *FirebaseManager) CreateDocument(collection string, docID string, data interface{}) error {
	_, err := fs.firebaseStore.Collection(collection).Doc(docID).Create(fs.ctx, data)
	return err
}

func (fs *FirebaseManager) CreateSubDocument(collection, docID, subcollection string, subDocID string, data interface{}) error {
	_, err := fs.firebaseStore.Collection(collection).Doc(docID).Collection(subcollection).Doc(subDocID).Create(fs.ctx, data)
	return err
}

func (fs *FirebaseManager) GetSubDocument(collection, docID, subcollection, subDocID string) (*firestore.DocumentSnapshot, error) {
	doc, err := fs.firebaseStore.Collection(collection).Doc(docID).Collection(subcollection).Doc(subDocID).Get(fs.ctx)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func (fs *FirebaseManager) GetSubDocuments(collection, docID, subcollection string) ([]map[string]interface{}, error) {
	iter := fs.firebaseStore.Collection(collection).Doc(docID).Collection(subcollection).Documents(fs.ctx)
	defer iter.Stop()

	var docs []map[string]interface{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc.Data())
	}
	return docs, nil
}

func (fs *FirebaseManager) UpdateSubDocument(collection, docID, subcollection, subDocID string, updates []firestore.Update) error {
	_, err := fs.firebaseStore.Collection(collection).Doc(docID).Collection(subcollection).Doc(subDocID).Update(fs.ctx, updates)
	return err
}

func (fs *FirebaseManager) DeleteSubDocument(collection, docID, subcollection, subDocID string) error {
	_, err := fs.firebaseStore.Collection(collection).Doc(docID).Collection(subcollection).Doc(subDocID).Delete(fs.ctx)
	return err
}

func (fs *FirebaseManager) GetDocument(collection string, docID string) (*firestore.DocumentSnapshot, error) {
	doc, err := fs.firebaseStore.Collection(collection).Doc(docID).Get(fs.ctx)
	if err != nil {
		return nil, err
	}
	return doc, nil
} // [END firestore_setup_client_get_document]

func (fs *FirebaseManager) GetDocuments(collection string) ([]map[string]interface{}, error) {
	iter := fs.firebaseStore.Collection(collection).Documents(fs.ctx)
	defer iter.Stop() // Close the iterator when done
	var docs []map[string]interface{}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		docs = append(docs, doc.Data())
		// fmt.Println(doc.Data())
		// fmt.Println(doc.Data()["Bin"])
	}
	return docs, nil
} // [END firestore_setup_client_get_documents]

func (fs *FirebaseManager) GetDocumentsTo(collection string, dataTo interface{}) ([]interface{}, error) {
	iter := fs.firebaseStore.Collection(collection).Documents(fs.ctx)
	defer iter.Stop() // Close the iterator when done
	var docs []interface{}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		docs = append(docs, doc.DataTo(&dataTo))
		// fmt.Println(doc.Data())
		// fmt.Println(doc.Data()["Bin"])
	}
	return docs, nil
} //

func (fs *FirebaseManager) GetSubDocumentsWhere(collection, docID, subcollection string, field string, op string, value interface{}) ([]map[string]interface{}, error) {
	iter := fs.firebaseStore.Collection(collection).Doc(docID).Collection(subcollection).Where(field, op, value).Documents(fs.ctx)
	defer iter.Stop()

	var docs []map[string]interface{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc.Data())
	}
	return docs, nil
}

func (fs *FirebaseManager) UpdateDocument(collection string, docID string, updates []firestore.Update) error {
	_, err := fs.firebaseStore.Collection(collection).Doc(docID).Update(fs.ctx, updates)
	return err
}

func (fs *FirebaseManager) DeleteDocument(collection string, docID string) error {
	_, err := fs.firebaseStore.Collection(collection).Doc(docID).Delete(fs.ctx)
	return err
}

// Bulk Operations
func (fs *FirebaseManager) BulkCreateDocuments(collection string, data []map[string]interface{}) error {
	bulkWriter := fs.firebaseStore.BulkWriter(fs.ctx)
	defer bulkWriter.Flush()

	for _, docData := range data {
		docRef := fs.firebaseStore.Collection(collection).NewDoc()
		_, err := bulkWriter.Create(docRef, docData)
		if err != nil {
			return err
		}
	}
	return nil
}
func (fs *FirebaseManager) BulkSetDocuments(collection string, data map[string]interface{}) error {
	bulkWriter := fs.firebaseStore.BulkWriter(fs.ctx)
	defer bulkWriter.Flush()

	for docID, docData := range data {
		docRef := fs.firebaseStore.Collection(collection).Doc(docID)
		_, err := bulkWriter.Set(docRef, docData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fs *FirebaseManager) BulkUpdateDocuments(collection string, updates map[string][]firestore.Update) error {
	bulkWriter := fs.firebaseStore.BulkWriter(fs.ctx)
	defer bulkWriter.Flush()

	for docID, updateData := range updates {
		docRef := fs.firebaseStore.Collection(collection).Doc(docID)
		_, err := bulkWriter.Update(docRef, updateData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fs *FirebaseManager) BulkDeleteDocuments(collection string, docIDs []string) error {
	bulkWriter := fs.firebaseStore.BulkWriter(fs.ctx)
	defer bulkWriter.Flush()

	for _, docID := range docIDs {
		docRef := fs.firebaseStore.Collection(collection).Doc(docID)
		_, err := bulkWriter.Delete(docRef)
		if err != nil {
			return err
		}
	}
	return nil
}

// Bulk Operations bypassing different collection from the data interface

func (fs *FirebaseManager) BulkSetDocumentsByPass(collection []map[string]interface{}, data []map[string]interface{}) error {
	bulkWriter := fs.firebaseStore.BulkWriter(fs.ctx)
	defer bulkWriter.Flush()

	// Iterate over userData and collection together
	for i := 0; i < len(data); i++ {
		docData := data[i]
		path := collection[i]["path"].(string) // Access collection name

		docRef := fs.firebaseStore.Collection(path).Doc(docData["id"].(string))
		_, err := bulkWriter.Set(docRef, docData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fs *FirebaseManager) Ctx() context.Context {
	return fs.ctx // Return the Firestore client
}

func (fs *FirebaseManager) GetCollectionByName(table interface{}) string {
	t := reflect.TypeOf(table)
	return strings.ToLower(t.Name())
}

func (fs *FirebaseManager) Close() {
	fs.firebaseStore.Close()
}

func (fs *FirebaseManager) Client() *firestore.Client {
	return fs.firebaseStore
}

func (fs *FirebaseManager) AuthClient() *auth.Client {
	return fs.firebaseAuth
}

func storeSecretFirebaseCredsAsFile() (string, error) {
	// Get the secret from environment variables
	encodedCreds := os.Getenv("FIREBASE_CREDENTIALS")
	if encodedCreds == "" {
		_, err := os.Open(os.Getenv("FILE_FIREBASE_CREDENTIALS"))
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("File does not exist")
			} else {
				log.Fatal(err)
			}

			fmt.Println("FIREBASE_CREDENTIALS not set")
			return "", fmt.Errorf("FIREBASE_CREDENTIALS not set")
		}

		return os.Getenv("FILE_FIREBASE_CREDENTIALS"), nil
	}

	// Decode Base64
	decoded, err := base64.StdEncoding.DecodeString(encodedCreds)
	if err != nil {
		fmt.Println("Error decoding Firebase credentials:", err)
		return "", fmt.Errorf("Error decoding Firebase credentials: %v", err)
	}

	// Save as a JSON file
	filePath := "/tmp/" + os.Getenv("FILE_FIREBASE_CREDENTIALS")
	err = os.WriteFile(filePath, decoded, 0600)
	if err != nil {
		fmt.Println("Error writing Firebase credentials file:", err)
		return "", fmt.Errorf("Error writing Firebase credentials file: %v", err)
	}

	fmt.Println("Firebase credentials saved at:", filePath)

	// Verify File Integrity (Calculate Hash)
	hash := sha256.Sum256(decoded)
	expectedHash := os.Getenv("FIREBASE_CREDENTIALS_HASH") // Retrieve expected hash from env
	if expectedHash != "" && fmt.Sprintf("%x", hash) != expectedHash {
		fmt.Println("Firebase credentials hash mismatch!")
		return "", fmt.Errorf("Firebase credentials hash mismatch")
	}

	// Parse and Validate JSON
	var creds Credentials
	err = json.Unmarshal(decoded, &creds)
	if err != nil {
		fmt.Println("Error unmarshaling Firebase credentials:", err)
		return "", fmt.Errorf("Error unmarshaling Firebase credentials: %v", err)
	}

	if creds.Type != "service_account" {
		return "", fmt.Errorf("invalid credential type: %s", creds.Type)
	}

	if creds.ProjectID == "" {
		return "", fmt.Errorf("project_id is required")
	}

	return filePath, nil
}
