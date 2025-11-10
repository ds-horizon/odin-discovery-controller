package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	idgenerator "odin-service-discovery-controller/internal/id-generator"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const uri = "/v1/record"
const httpPrefix = "http://"
const retryCount = 3

var idGenerator = idgenerator.NewIDGenerator()

var (
	client        *http.Client
	mutex         sync.Mutex
	clientCreated bool
)

// Record : struct for the discovery record
type Record struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// Request : struct for the request to discovery service
type Request struct {
	Action string `json:"action"`
	Record Record `json:"record"`
	ID     string `json:"id"`
}

// Webclient : the webclient for sending requests to discovery service
type Webclient struct {
	backendAddress string
	batchDuration  time.Duration
	batchSize      int
	requestChannel chan Request
	orgID          string
	accountName    string
}

// CreateWebClient : Function for creating the webclient
func CreateWebClient(backendAddress string, batchDuration time.Duration, batchSize int, orgID string, accountName string) *Webclient {
	return &Webclient{
		backendAddress: backendAddress,
		batchDuration:  batchDuration,
		batchSize:      batchSize,
		requestChannel: make(chan Request, batchSize),
		orgID:          orgID,
		accountName:    accountName,
	}
}

// AddToBatch : Function for adding request to the batch
func (c *Webclient) AddToBatch(request Request) {
	request.ID = strconv.FormatInt(idGenerator.GetNextID(), 10) // Use the next ID
	log.Infof("Adding request to batch: %v\n", request)
	c.requestChannel <- request
}

// Run : Function for triggering the processing of the  batch
func (c *Webclient) Run() {
	go c.processBatch()
}

func (c *Webclient) processBatch() {
	var batch []Request
	ackChannel := make(chan string, 10)
	timer := time.NewTimer(c.batchDuration)
	for {
		select {
		case req := <-c.requestChannel:
			batch = append(batch, req)
			if len(batch) >= c.batchSize {
				log.Infof("Size threshold has been reached, sending batch to discovery service: %v\n", batch)
				reqBody, err := c.formRequestBody(batch)
				if err != nil {
					log.Errorf("error in unmarshalling json: %s", err)
				}
				err = c.sendBatchToDiscoveryService(reqBody, ackChannel)
				if err != nil {
					log.Error("Error sending batch to backend:", err)
				}
				batch = nil
				if !timer.Stop() {
					<-timer.C
				}
				<-ackChannel //block until you get the acknowledgement for the current batch
				log.Infof("Acknowledgement received for batch")
				timer.Reset(c.batchDuration)
			}
		case <-timer.C:
			if len(batch) > 0 {
				log.Infof("Time threshold has been reached, sending batch to discovery service: %v\n", batch)
				reqBody, err := c.formRequestBody(batch)
				if err != nil {
					log.Errorf("error in unmarshalling json: %s", err)
				}
				err = c.sendBatchToDiscoveryService(reqBody, ackChannel)
				if err != nil {
					log.Error("Error sending batch to backend:", err)
				}
				batch = nil
				<-ackChannel
				log.Infof("Acknowledgement received for batch")
			}
			timer.Reset(c.batchDuration)
		}
	}
}

func getClient() *http.Client {
	if !clientCreated {
		log.Info("starting HTTP client")
		mutex.Lock()
		defer mutex.Unlock()
		client = &http.Client{}
		clientCreated = true
	}
	return client
}

func (c *Webclient) formRequestBody(batch []Request) ([]byte, error) {
	requestData := struct {
		AccountName string    `json:"accountName"`
		Batch       []Request `json:"recordActions"`
	}{
		AccountName: c.accountName,
		Batch:       batch,
	}

	log.Infof("Request body generated: %v\n", requestData)

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}
	return requestBody, nil
}

func getURL(backendAddress string) string {
	return httpPrefix + backendAddress + uri
}

func (c *Webclient) sendBatchToDiscoveryService(requestBody []byte, ackChannel chan<- string) error {

	defer func() {
		ackChannel <- "Ack for this batch"
	}()

	// Use a separate channel for retry notifications
	retryChannel := make(chan bool)

	// Start a goroutine for asynchronous retries
	go func() {
		for i := 0; i < retryCount; i++ {
			// Wait for a signal on the retry channel
			<-retryChannel

			log.Infof("Retrying request, attempt %d", i+1)

			// Create a new request with the same data
			req, err := http.NewRequest("PUT", getURL(c.backendAddress), bytes.NewBuffer(requestBody))
			if err != nil {
				log.Error("Error creating retry request:", err)
				continue
			}

			// Set headers
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("OrgId", c.orgID)

			// Send the retry request
			resp, err := getClient().Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				log.Info("Retry successful")
				close(retryChannel) // Close the retry channel to signal success
				return
			}
		}

		log.Error("Exhausted all retries, giving up")
		close(retryChannel) // Close the retry channel to signal failure
	}()

	// Create a custom request
	req, err := http.NewRequest("PUT", getURL(c.backendAddress), bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OrgId", c.orgID)

	log.Infof("Request to be sent to the backend: %v\n", req)

	// Send the HTTP request using a custom http.Client
	resp, err := getClient().Do(req)

	log.Infof("Response received: %v with status code %d", resp, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		retryChannel <- true
		log.Infof("Response from discovery service %v with status code %d", resp, resp.StatusCode)
		return err
	}

	if err != nil {
		// Notify the retry goroutine to retry
		retryChannel <- true
		log.Errorf("Error while submitting request to discovery service: %s", err)
		// Return without waiting for the retry
		return err
	}
	defer resp.Body.Close()

	log.Info("Request successfully submitted to discovery service")

	return nil
}
