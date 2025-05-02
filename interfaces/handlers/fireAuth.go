package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/deepscrape/arachnefly/domain"
	"github.com/deepscrape/arachnefly/domain/routine"
	"github.com/deepscrape/arachnefly/infrastructure/routines"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	machines "github.com/sosedoff/fly-machines"
)

// --- Configuration ---

const (
	// Use Fly.io's internal API endpoint for lower latency within the network [1]
	flyAPIEndpointInternal = "http://_api.internal:4280"
	flyAPITimeout          = 5 * time.Second  // Timeout for calls to the Fly Machines API
	proxyRequestTimeout    = 15 * time.Second // Overall timeout for handling a proxied request
	targetServicePort      = "11235"          // The port the target service listens on inside the tenant machine
)

type AuthHandler struct {
	globalRoutines routine.IGlobal
	// validator *validator.CustomValidator

	// AUTH_SESSION_NAME          string
	// OAUTH_STATE_COOKIE         string
	// OAUTH_CODE_VERIFIER_COOKIE string
	// OAUTH_CODE_CHAL_COOKIE     string
	// CSRF_SESSION_NAME          string

	// cfg    *config.Config
	// logger *zap.Logger
	flyApp    string
	flyApiUrl string
}

func NewHandlers(flyApiToken, flyApiUrl, flyApp string) *AuthHandler {

	return &AuthHandler{
		globalRoutines: routines.NewGlobalRoutines(flyApiToken),
		flyApp:         flyApp,
		flyApiUrl:      flyApiUrl,
		// cfg:                        cfg,
		// logger:                     logger,
	}
}

func (h *AuthHandler) GetMachine(c *fiber.Ctx) (*domain.HTTPResponse, error) {

	var response domain.HTTPResponse

	machineId := c.Params("id")

	if machineId == "" {
		response = domain.HTTPResponse{
			Code:    fiber.StatusBadRequest,
			Status:  "error",
			Message: "Machine ID is required",
			Errors:  domain.APIError{Code: "MachineIDRequired", Message: "Machine ID is required"},
			Data:    fiber.Map{"status": false},
		}
		return &response, nil
	}
	// 1. Get Machine Details
	machine, err := h.globalRoutines.GetMachineDetails(machineId, h.flyApiUrl, h.flyApp)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get machine details",
			Errors:  domain.APIError{Code: "GetMachineDetails Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	ip, ok := machine["private_ip"].(string)
	if !ok {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get machine private ip",
			Errors:  domain.APIError{Code: "GetMachine IP Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	// Resolve the internal hostname to an IPv6 address
	ips, err := net.LookupIP(ip)
	if err != nil || len(ips) == 0 {

		response = domain.HTTPResponse{
			Code:    fiber.StatusBadRequest,
			Status:  "error",
			Message: "DNS not found",
			Errors:  domain.APIError{Code: "DNSNotFound", Message: "DNS resolution failed"},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	return &domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Machine details",
		Data:    fiber.Map{"status": true, "message": "Machine details", "machine": machine, "ip": ips[0].String()},
	}, nil

}

// 🚀 Deploy Machine (Clone or New)
func (h *AuthHandler) DeployMachine(c *fiber.Ctx) (*domain.HTTPResponse, error) {

	clone := c.Query("clone") == "true"
	masterId := c.Query("master_id")
	region := c.Query("region")

	if region == "" || len(region) != 3 {
		fmt.Println("Warning: Region should be 3 letters. Using default.")
		region = "iad"
	}
	var config map[string]interface{}
	var response domain.HTTPResponse

	/* Clone Machine */
	if clone && masterId != "" {
		machine, err := h.globalRoutines.GetMachineDetails(masterId, h.flyApiUrl, h.flyApp)
		// log.Println("Machine:", machine)

		if err != nil {
			response = domain.HTTPResponse{
				// Headers: headers,
				Code:    fiber.StatusBadRequest,
				Status:  "error",
				Message: "Check the Machine Details again",
				Errors:  domain.APIError{Code: "Get Machine Details Failed", Message: err.Error()},
				Data:    fiber.Map{"status": false},
			}

			return &response, err
		}
		config = map[string]interface{}{
			"region": machine["region"],
			"config": machine["config"],
		}
	} else if !clone {
		// Parse the incoming JSON data
		var machineConfig domain.MachineConfig
		if err := c.BodyParser(&machineConfig); err != nil {
			response = domain.HTTPResponse{
				Code:    fiber.StatusBadRequest,
				Status:  "error",
				Message: "Failed to parse request body",
				Errors:  domain.APIError{Code: "Parse Error", Message: err.Error()},
				Data:    fiber.Map{"status": false},
			}
			return &response, err
		}

		// print machineconfig
		log.Println("MachineConfig:", machineConfig)

		config = h.globalRoutines.BuildConfigMap(machineConfig)
	}

	// This code snippet is handling the deployment of a machine using the `FlyRequest` method from the
	// `globalRoutines` object. Here's a breakdown of what's happening:
	machine, err := h.globalRoutines.FlyRequest("POST", fmt.Sprintf("%s/apps/%s/machines", h.flyApiUrl, h.flyApp), config, nil)
	if err != nil {
		log.Fatalf("Fly Request Error: %v", err)
		response = domain.HTTPResponse{
			// Headers: headers,
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Check the Fly api url or fly app image if exists",
			Errors:  domain.APIError{Code: "Deploy by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}

		return &response, err
	}

	response = domain.HTTPResponse{
		// Headers: headers,
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Machine deployed Successful",
		Data:    fiber.Map{"status": true, "message": "Machine deployed", "machine_id": machine["id"], "machine_details": machine},
	}

	return &response, nil

}

// 🚀 Check or Create Fly App
func (h *AuthHandler) CheckOrCreateApp(c *fiber.Ctx) error {
	appName := h.flyApp
	// var response domain.HTTPResponse

	/* if appName == "" {
		response = domain.HTTPResponse{
			Code:    fiber.StatusBadRequest,
			Status:  "error",
			Message: "App name is required",
			Errors:  domain.APIError{Code: "AppNameRequired", Message: "App name is required"},
			Data:    fiber.Map{"status": false},
		}
		return
	} */

	// Check if the app exists
	app, err := h.globalRoutines.FlyRequest("GET", fmt.Sprintf("%s/apps/%s", h.flyApiUrl, h.flyApp), nil, nil)
	if err != nil {

		return &fiber.Error{Code: fiber.StatusInternalServerError, Message: fmt.Sprintf("Failed to fetch apps from Fly.io %s", err.Error())}
	}

	// Check if the app name exists in the list
	if app["name"] == appName {
		log.Printf("App already exists: %s", app["name"])
		return c.Next()
	}

	if app["error"] != nil && app["error"] != "" {
		log.Println("App:", app["error"])
		// App does not exist, create a new app
		appConfig := map[string]interface{}{
			"app_name":          appName,
			"org_slug":          "softmind",
			"enable_subdomains": true,
			"network":           "dedicated",
		}

		newApp, err := h.globalRoutines.FlyRequest("POST", fmt.Sprintf("%s/apps", h.flyApiUrl), appConfig, nil)
		if err != nil {
			return &fiber.Error{Code: fiber.StatusInternalServerError, Message: fmt.Sprintf("Failed to create app on Fly.io: %s", err.Error())}
		}

		// return &response, nil
		if newApp["error"] != nil && newApp["error"] != "" {

			return &fiber.Error{Code: fiber.StatusInternalServerError, Message: fmt.Sprintf("Fly.io %s", newApp["error"])}
		}
	}

	log.Printf("New App may Created successfully")
	return c.Next()
}

// 🚀 Start Machine
func (h *AuthHandler) StartMachine(c *fiber.Ctx) (*domain.HTTPResponse, error) {
	machineId := c.Params("id")
	var response domain.HTTPResponse

	if machineId != "" {
		response = machineIdRequired()
		return &response, nil
	}

	// post request to start machine by id
	_, err := h.globalRoutines.FlyRequest("POST", fmt.Sprintf("%s/apps/%s/machines/%s/start", h.flyApiUrl, h.flyApp, machineId), nil, nil)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to start machine",
			Errors:  domain.APIError{Code: "StartMachine by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}

		return &response, err
	}
	response = domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Machine started",
		Data:    fiber.Map{"status": true, "message": "Machine started"},
	}
	return &response, nil
}

// 🚀 Stop Machine
func (h *AuthHandler) StopMachine(c *fiber.Ctx) (*domain.HTTPResponse, error) {
	machineId := c.Params("id")
	var response domain.HTTPResponse

	if machineId != "" {
		response = machineIdRequired()
		return &response, nil
	}
	_, err := h.globalRoutines.FlyRequest("POST", fmt.Sprintf("%s/apps/%s/machines/%s/stop", h.flyApiUrl, h.flyApp, machineId), nil, nil)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to stop the machine by id " + machineId,
			Errors:  domain.APIError{Code: "StopMachine by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}

		return &response, err
	}

	response = domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Machine stopped",
		Data:    fiber.Map{"status": true, "message": "Machine stopped"},
	}
	return &response, nil
}

// 🚀 Delete Machine
func (h *AuthHandler) DeleteMachine(c *fiber.Ctx) (*domain.HTTPResponse, error) {
	machineId := c.Params("id")
	var response domain.HTTPResponse
	if machineId != "" {
		response = machineIdRequired()
		return &response, nil
	}
	_, err := h.globalRoutines.FlyRequest("DELETE", fmt.Sprintf("%s/apps/%s/machines/%s", h.flyApiUrl, h.flyApp, machineId), nil, nil)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to delete the machine by id " + machineId,
			Errors:  domain.APIError{Code: "DeleteMachine by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}
	response = domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Machine deleted",
		Data:    fiber.Map{"status": true, "message": "Machine deleted"},
	}
	return &response, nil
}

// 🚀 Execute Task on Running Machine
func (h *AuthHandler) ExecuteTask(c *fiber.Ctx) (*domain.HTTPResponse, error) {
	machineId := c.Params("machine_id")
	var response domain.HTTPResponse

	log.Println("Machine ID:", machineId)
	if machineId == "" {
		response = machineIdRequired()
		return &response, nil
	}

	var MachineExecute map[string]interface{}
	if err := c.BodyParser(&MachineExecute); err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to parse command",

			Errors: domain.APIError{Code: "ExecuteTask Failed", Message: err.Error()},
			Data:   fiber.Map{"status": false},
		}
		return &response, err
	}

	// json.Unmarshal(MachineExecute.Command, &MachineExecute.Command)

	machine, err := h.globalRoutines.GetMachineDetails(machineId, h.flyApiUrl, h.flyApp)
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get machine details",
			Errors:  domain.APIError{Code: "GetMachineDetails by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	client := machines.NewClient(h.flyApp)
	// client.SetBaseURL(machines.PrivateBaseURL)

	var listMachines []machines.Machine
	listMachines, err = client.List(&machines.ListInput{
		State: "running",
	})

	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get list of machines",
			Errors:  domain.APIError{Code: "ListMachines by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	for _, machine := range listMachines {
		log.Println("Machine Name:", machine.Name, "Machine ID:", machine.ID, "Machine IP:", machine.PrivateIP, "Machine State:", machine.State, "Machine Region:", machine.Region, machine.ImageRef.Registry, machine.ImageRef.Digest, machine.ImageRef.Repository, machine.ImageRef.Tag)
	}

	_, ok := machine["private_ip"].(string)
	if !ok {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to get machine IP",
			Errors:  domain.APIError{Code: "GetMachineIP by FlyRequest Failed"},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}
	taskUrl := fmt.Sprintf("https://%s/crawl", h.flyApp+".fly.dev")

	log.Println("Task URL:", taskUrl, MachineExecute)

	data, err := h.globalRoutines.FlyRequest("POST", taskUrl, MachineExecute, map[string]string{"fly-replay": fmt.Sprintf("instance=%s", machineId)})
	if err != nil {
		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Failed to execute task on machine",
			Errors:  domain.APIError{Code: "ExecuteTask by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	response = domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Task executed on machine",
		Data:    fiber.Map{"status": true, "message": "Task executed on machine", "machine_data": data},
	}
	return &response, nil
}

// handleContainerProxyFiber handles incoming requests using Fiber, authenticates,
// finds the target machine, verifies its state via the Fly API, and proxies
// the request directly over 6PN using Fiber's proxy middleware.
func (h *AuthHandler) HandleContainerProxyFiber(c *fiber.Ctx) (*domain.HTTPResponse, error) {
	var response domain.HTTPResponse
	var targetAppName string = h.flyApp

	machineId := c.Params("id")

	if machineId == "" {
		response = machineIdRequired()
		return &response, nil
	}

	// Set an overall timeout for the request handling using the Fiber context
	// Note: Fiber uses UserContext() to bridge to standard context.Context
	// ctx, cancel := context.WithTimeout(c.UserContext(), proxyRequestTimeout)
	// defer cancel()

	// log.Printf("INFO: User %s mapped to App: %s, MachineID: %s", userID, targetAppName, targetMachineID)

	// (3c) Get target machine state and details from Fly Machines API
	// Pass the derived context with timeout
	machine, err := h.globalRoutines.GetMachineDetails(machineId, h.flyApiUrl, targetAppName)
	if err != nil {
		log.Printf("ERROR: Fly Machine API call failed for %s/%s: %v", targetAppName, machineId, err)

		response = domain.HTTPResponse{
			Code:    fiber.StatusInternalServerError,
			Status:  "error",
			Message: "Target machine not found or inaccessible",
			Errors:  domain.APIError{Code: "GetMachineDetails by FlyRequest Failed", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	// (3d) Check if the machine state is 'started' [1, 2]
	if machine["state"] != "started" {
		log.Printf("WARN: Target machine %s/%s is not started (state: %s). Cannot proxy request for user %s.", targetAppName, machine["id"], machine["state"], "userID")
		response = domain.HTTPResponse{
			Code:    fiber.StatusBadRequest,
			Status:  "error",
			Message: "Target machine is not in a running state",
			Errors:  domain.APIError{Code: "MachineNotRunning", Message: fmt.Sprintf("Target machine is not running (state: %s)", machine["state"])},
			Data:    fiber.Map{"status": false},
		}
		return &response, nil
	}
	log.Printf("INFO: Target machine %s/%s confirmed as 'started'", targetAppName, machine["id"])

	// (3e) Construct the target machine's stable internal DNS hostname [3, 4]
	targetHost := fmt.Sprintf("%s.vm.%s.internal", machine["id"], targetAppName)

	// (f & g) Construct the target URL for Fiber's proxy
	targetURL := &url.URL{
		Scheme: "http", // Assuming target service listens on HTTP internally
		Host:   net.JoinHostPort(targetHost, targetServicePort),
	}
	targetAddr := targetURL.String() // Proxy functions need the address string

	log.Printf("INFO: Proxying request for user %s to target URL: %s", "userID", targetAddr)

	/* ips, err := net.LookupIP("fdaa:b:1166:a7b:f6:3351:7336:2")
	if err != nil || len(ips) == 0 {

		response = domain.HTTPResponse{
			Code:    fiber.StatusBadRequest,
			Status:  "error",
			Message: "DNS not found",
			Errors:  domain.APIError{Code: "DNSNotFound", Message: "DNS resolution failed"},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	log.Println("IPs:", ips) */

	// Set modified headers before proxying (IMPORTANT!)
	// Remove any sensitive headers if needed
	c.Request().Header.Del("Authorization")
	// c.Request().Header.Del("Cookie")

	// Add headers to provide context to the backend service
	c.Request().Header.Set("X-Forwarded-Host", string(c.Request().Host()))
	c.Request().Header.Set("X-Authenticated-User-Id", "userID")

	// Use c.IP() for the client IP
	// Add X-Forwarded-For if not already set by Fly Proxy (Fly adds Fly-Client-IP [5])
	if c.Request().Header.Peek("X-Forwarded-For") == nil {
		c.Request().Header.Set("X-Forwarded-For", c.IP())
	}

	// CRITICAL: Set the Host header for the target service
	c.Request().Header.Set("Host", targetHost)

	// Execute the proxy forward with the modifier
	// proxy.Do forwards the request and copies the response back to c.Response()
	// It requires the target address string.
	// Use Fiber's proxy middleware to forward the request
	if err := proxy.Do(c, targetAddr); err != nil {
		log.Printf("ERROR: Proxy error to %s: %v", targetURL, err)

		// Handle different error types
		var netErr *net.OpError
		if errors.As(err, &netErr) && netErr.Op == "dial" {
			response = domain.HTTPResponse{
				Code:    fiber.StatusBadGateway,
				Status:  "error",
				Message: "Bad Gateway: Cannot connect to target service",
				Errors:  domain.APIError{Code: "NetworkError", Message: err.Error()},
				Data:    fiber.Map{"status": false},
			}
			return &response, err
		} else if errors.Is(err, context.DeadlineExceeded) {
			// Generic proxy error
			response = domain.HTTPResponse{
				Code:    fiber.StatusGatewayTimeout,
				Status:  "error",
				Message: "Bad Gateway: Gateway Timeout",
				Errors:  domain.APIError{Code: "ProxyError", Message: err.Error()},
				Data:    fiber.Map{"status": false},
			}
			return &response, err
		}

		// Generic proxy error
		response = domain.HTTPResponse{
			Code:    fiber.StatusBadGateway,
			Status:  "error",
			Message: "Bad Gateway: Proxy error",
			Errors:  domain.APIError{Code: "ProxyError", Message: err.Error()},
			Data:    fiber.Map{"status": false},
		}
		return &response, err
	}

	response = domain.HTTPResponse{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: "Request proxied successfully",
		Errors:  domain.APIError{Code: "ProxySuccess", Message: "Request proxied successfully"},
		Data:    fiber.Map{"status": true},
	}
	// If proxy.Do returns nil, the response has already been written to c.Response()
	return &response, nil // Indicate successful handling by the proxy
}

func machineIdRequired() domain.HTTPResponse {

	return domain.HTTPResponse{
		Code:    fiber.StatusBadRequest,
		Status:  "error",
		Message: "Machine ID is required",
		Errors:  domain.APIError{Code: "Machine ID is required", Message: "Machine ID is required"},
		Data:    fiber.Map{"status": false},
	}
}
