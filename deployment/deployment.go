package deployment

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
)

func FetchAndWriteDeploymentData(inputURL, clusterName, window, bucketName, region string, wg *sync.WaitGroup) {
	u, err := url.Parse(inputURL)
	if err != nil {
		configs.ErrorLogger.Println("Error parsing URL:", err)
		return
	}

	u.Path = "/model/allocation"
	q := u.Query()
	q.Set("window", window)
	q.Set("aggregate", "deployment")
	q.Set("accumulate", "true")
	u.RawQuery = q.Encode()

	newURL := u.String()

	resp, err := http.Get(newURL)
	if err != nil {
		configs.ErrorLogger.Println("Error making HTTP request:", err)
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
	configs.InfoLogger.Println("Status Code for Deployment:", result["code"])

	data := result["data"].([]interface{})


	svc := configs.Svc

	objectKey := "Deployment/Deployment.csv"

	
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
		configs.InfoLogger.Println("No existing Deployment.csv file found. A new one will be created.")
	}

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	defer writer.Flush()

	
	if !fileExists {
		header := []string{
			"Deployment", "ClusterName", "Region", "Namespace",
			"Window Start", "Window End", "Cpu Cost", "Gpu Cost",
			"Ram Cost", "PV Cost", "Network Cost", "LoadBalancer Cost",
			"Shared Cost", "Total Cost", "Cpu Efficiency", "Ram Efficiency", "Total Efficiency",
		}
		if err := writer.Write(header); err != nil {
			configs.ErrorLogger.Println("Error writing header to CSV:", err)
			return
		}
	}

	
	for _, element := range data {
		deploymentMap := element.(map[string]interface{})

		for _, DeploymentData := range deploymentMap {
			deploymentOne := DeploymentData.(map[string]interface{})

			name := deploymentOne["name"].(string)
			if name == "__unallocated__" {
				continue
			}

			properties := deploymentOne["properties"].(map[string]interface{})
			labels := properties["labels"].(map[string]interface{})
			region := labels["topology_kubernetes_io_region"].(string)

			namespaceDeployment, ok := labels["kubernetes_io_metadata_name"].(string)
			if !ok {
				namespaceDeployment = ""
			}

			window := deploymentOne["window"].(map[string]interface{})
			windowStart := window["start"].(string)
			windowEnd := window["end"].(string)

			cpuCost := deploymentOne["cpuCost"].(float64)
			gpuCost := deploymentOne["gpuCost"].(float64)
			ramCost := deploymentOne["ramCost"].(float64)
			pvCost := deploymentOne["pvCost"].(float64)
			networkCost := deploymentOne["networkCost"].(float64)
			loadBalancerCost := deploymentOne["loadBalancerCost"].(float64)
			sharedCost := deploymentOne["sharedCost"].(float64)
			totalCost := deploymentOne["totalCost"].(float64)
			cpuEfficiency := deploymentOne["cpuEfficiency"].(float64) * 100
			ramEfficiency := deploymentOne["ramEfficiency"].(float64) * 100
			totalEfficiency := deploymentOne["totalEfficiency"].(float64) * 100

			record := []string{
				name, clusterName, region, namespaceDeployment, windowStart, windowEnd,
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

	configs.InfoLogger.Println("Deployment data successfully written to S3")
	wg.Done()
}
