.PHONY: set-firebase-secret set-fly-api-token

# Include the environment file if it exists
-include .env

# Define the path to your Firebase secury account credentials JSON file
FILE_FIREBASE_CREDENTIALS ?=./libnet-d76db-949683c2222d.json

# Command to encode JSON as Base64 and set it as a Fly.io secret
set-firebase-secret:
	@echo "Setting Firebase credentials in Fly.io..."
	@[ -f $(FILE_FIREBASE_CREDENTIALS) ] || { echo "Error: Firebase JSON file not found at $(FILE_FIREBASE_CREDENTIALS)"; exit 1; }
	@flyctl secrets set FIREBASE_CREDENTIALS="$$(base64 < $(FILE_FIREBASE_CREDENTIALS))"
	@echo "✅ Firebase credentials set successfully!"

set-fly-api-token:
	@echo "Setting Fly.io API token..."
	@flyctl secrets set FLY_API_TOKEN="$$(flyctl auth token)"
	@echo "✅ Fly.io API token set successfully!"

set-firebase-hash:
	@echo "Setting Firebase credentials hash in Fly.io..."
	@[ -f $(FILE_FIREBASE_CREDENTIALS) ] || { echo "Error: Firebase JSON file not found at $(FILE_FIREBASE_CREDENTIALS)"; exit 1; }
	ifeq ($(OS),Windows_NT)
		FIREBASE_HASH := $$(powershell -Command "Get-FileHash -Algorithm SHA256 -Path $(FILE_FIREBASE_CREDENTIALS) | Select-Object -ExpandProperty Hash")
	else
		FIREBASE_HASH := $$(openssl dgst -sha256 -r $(FILE_FIREBASE_CREDENTIALS) | awk '{print $$1}')
	endif
	@flyctl secrets set FIREBASE_CREDENTIALS_HASH="$${FIREBASE_HASH}"
	@echo "✅ Firebase credentials hash set successfully!"
