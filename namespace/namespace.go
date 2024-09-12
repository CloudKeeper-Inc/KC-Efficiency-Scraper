package namespace

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"kubecost-efficiency-fetcher/configs"
	"net/http"
	"net/url"
	"sync"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"time"
)

func FetchAndWriteNamespaceData(inputURL, clusterName, window, bucketName string, wg *sync.WaitGroup) {

	defer wg.Done()
	u, err := url.Parse(inputURL)
	if err != nil {
		configs.ErrorLogger.Println("Error parsing URL:", err)
		return
	}

	u.Path = "/model/allocation"
	q := u.Query()
	q.Set("window", window)
	q.Set("aggregate", "namespace")
	q.Set("accumulate", "true")
	u.RawQuery = q.Encode()

	newURL := u.String()

	var resp *http.Response
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err = http.Get(newURL)
		if err == nil {
			break
		}
		configs.ErrorLogger.Printf("Attempt %d: Error making HTTP request: %v\n", attempt, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		configs.ErrorLogger.Println("Failed to make HTTP request after multiple attempts:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		configs.ErrorLogger.Println("Error reading response body:", err)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		configs.ErrorLogger.Println("Error unmarshalling JSON:", err)
		return
	}

	configs.InfoLogger.Println("Status Code for Namespace:", result["code"])

	data := result["data"].([]interface{})


	svc := configs.Svc

	objectKey := "Namespace/Namespace.csv"

	existingData := [][]string{}
	fileExists := false

	_, err = svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err == nil {

		resp, err := svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
		})
		if err != nil {
			configs.ErrorLogger.Println("Error fetching existing file from S3:", err)
			return
		}
		defer resp.Body.Close()

		reader := csv.NewReader(resp.Body)
		existingData, err = reader.ReadAll()
		if err != nil {
			configs.ErrorLogger.Println("Error reading existing CSV data:", err)
			return
		}
		fileExists = true
	} else {
		configs.InfoLogger.Println("No existing Namespace.csv file found. A new one will be created.")
	}

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	defer writer.Flush()


	if !fileExists {
		header := []string{"Namespace","ClusterName", "Region", "Window Start", "Window End", "Cpu Cost", "Gpu Cost", "Ram Cost", "PV Cost", "Network Cost", "LoadBalancer Cost", "Shared Cost", "Total Cost", "Cpu Efficiency", "Ram Efficiency", "Total Efficiency"}
		if err := writer.Write(header); err != nil {
			configs.ErrorLogger.Println("Error writing header to CSV:", err)
			return
		}
	}


	for _, element := range data {
		if element == nil{
			configs.InfoLogger.Println("No Data for Namespace")
			continue
		}
		namespaceMap := element.(map[string]interface{})

		for _, namespaceData := range namespaceMap {
			namespaceOne := namespaceData.(map[string]interface{})

			name := namespaceOne["name"].(string)
			if name == "__unallocated__" {
				continue
			}

			properties := namespaceOne["properties"].(map[string]interface{})
			
			var labels map[string]interface{}
			var region string
			var namespace string
			
			if value, ok := properties["labels"].(map[string]interface{}); ok {
				labels = value
				if val, ok := labels["topology_kubernetes_io_region"].(string); ok {
					region = val
				} else {
					region = "" 
				}
			
				if val, ok := labels["kubernetes_io_metadata_name"].(string); ok {
					namespace = val
				} else {
					namespace = "" 
				}
			} else {
				region = ""
				namespace = ""
			}

			window := namespaceOne["window"].(map[string]interface{})
			windowStart := window["start"].(string)
			windowEnd := window["end"].(string)

			cpuCost := namespaceOne["cpuCost"].(float64)
			gpuCost := namespaceOne["gpuCost"].(float64)
			ramCost := namespaceOne["ramCost"].(float64)
			pvCost := namespaceOne["pvCost"].(float64)
			networkCost := namespaceOne["networkCost"].(float64)
			loadBalancerCost := namespaceOne["loadBalancerCost"].(float64)
			sharedCost := namespaceOne["sharedCost"].(float64)
			totalCost := namespaceOne["totalCost"].(float64)
			cpuEfficiency := namespaceOne["cpuEfficiency"].(float64) * 100
			ramEfficiency := namespaceOne["ramEfficiency"].(float64) * 100
			totalEfficiency := namespaceOne["totalEfficiency"].(float64) * 100

			record := []string{
				namespace,clusterName, region, windowStart, windowEnd,
				fmt.Sprintf("%f", cpuCost), fmt.Sprintf("%f", gpuCost),
				fmt.Sprintf("%f", ramCost), fmt.Sprintf("%f", pvCost),
				fmt.Sprintf("%f", networkCost), fmt.Sprintf("%f", loadBalancerCost),
				fmt.Sprintf("%f", sharedCost), fmt.Sprintf("%f", totalCost),
				fmt.Sprintf("%f", cpuEfficiency), fmt.Sprintf("%f", ramEfficiency),
				fmt.Sprintf("%f", totalEfficiency),
			}
			existingData = append(existingData, record)
		}
	}


	if err := writer.WriteAll(existingData); err != nil {
		configs.ErrorLogger.Println("Error writing data to CSV:", err)
		return
	}


	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(buffer.Bytes()),
		ContentType: aws.String("text/csv"),
	})
	if err != nil {
		configs.ErrorLogger.Println("Error uploading updated CSV to S3:", err)
		return
	}

	configs.InfoLogger.Println("Namespace data successfully written to S3")
}
