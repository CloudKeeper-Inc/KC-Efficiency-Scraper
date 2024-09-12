package cluster

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
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func FetchAndWriteClusterData(inputURL,clusterName, window, bucketName, region string, wg *sync.WaitGroup) {

	defer wg.Done()
	u, err := url.Parse(inputURL)
	if err != nil {
		configs.ErrorLogger.Println("Error parsing URL:", err)
		return
	}
	u.Path = "/model/allocation"
	q := u.Query()
	q.Set("window", window)
	q.Set("aggregate", "cluster")
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
	configs.InfoLogger.Println("Status Code for Cluster: ", result["code"])


	data := result["data"].([]interface{})


	svc := configs.Svc


	objectKey := "Cluster/Cluster.csv"


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

		configs.InfoLogger.Println("No existing Cluster.csv file found. A new one will be created.")
	}


	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	defer writer.Flush()


	if !fileExists {
		header := []string{"Cluster", "Window Start", "Window End", "Cpu Cost", "Gpu Cost", "Ram Cost", "PV Cost", "Network Cost", "LoadBalancer Cost", "Shared Cost", "Total Cost", "Cpu Efficiency", "Ram Efficiency", "Total Efficiency"}
		if err := writer.Write(header); err != nil {
			configs.ErrorLogger.Println("Error writing header to CSV:", err)
			return
		}
	}

	for _, element := range data {
		if element == nil{
			continue
		}
		clusterMap := element.(map[string]interface{})

		for _, clusterData := range clusterMap {
			clusterOne := clusterData.(map[string]interface{})

			properties := clusterOne["properties"].(map[string]interface{})
			cluster := properties["cluster"].(string)
			if(cluster == "cluster-one"){
				cluster = clusterName
			}


			window := clusterOne["window"].(map[string]interface{})
			windowStart := window["start"].(string)
			windowEnd := window["end"].(string)

			cpuCost := clusterOne["cpuCost"].(float64)
			gpuCost := clusterOne["gpuCost"].(float64)
			ramCost := clusterOne["ramCost"].(float64)
			pvCost := clusterOne["pvCost"].(float64)
			networkCost := clusterOne["networkCost"].(float64)
			loadBalancerCost := clusterOne["loadBalancerCost"].(float64)
			sharedCost := clusterOne["sharedCost"].(float64)
			totalCost := clusterOne["totalCost"].(float64)
			cpuEfficiency := clusterOne["cpuEfficiency"].(float64) * 100
			ramEfficiency := clusterOne["ramEfficiency"].(float64) * 100
			totalEfficiency := clusterOne["totalEfficiency"].(float64) * 100

			record := []string{
				cluster, windowStart, windowEnd,
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

	configs.InfoLogger.Println("Cluster data successfully written to S3")
}