package controllerKind

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

func FetchAndWriteControllerKindData(inputURL,clusterName, window, bucketName, region string, wg *sync.WaitGroup) {


	u, err := url.Parse(inputURL)
	if err != nil {
		configs.ErrorLogger.Println("Error parsing URL:", err)
		return
	}

	u.Path = "/model/allocation"
	q := u.Query()
	q.Set("window", window)
	q.Set("aggregate", "controllerKind")
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
	configs.InfoLogger.Println("Status Code for ControllerKind: ", result["code"])

	data := result["data"].([]interface{})



	svc := configs.Svc

	objectKey := "ControllerKind/ControllerKind.csv"


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
		configs.InfoLogger.Println("No existing ControllerKind.csv file found. A new one will be created.")
	}


	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	defer writer.Flush()


	if !fileExists {
		header := []string{"ControllerKind","ClusterName", "Region", "Window Start", "Window End", "Cpu Cost", "Gpu Cost", "Ram Cost", "PV Cost", "Network Cost", "LoadBalancer Cost", "Shared Cost", "Total Cost", "Cpu Efficiency", "Ram Efficiency", "Total Efficiency"}
		if err := writer.Write(header); err != nil {
			configs.ErrorLogger.Println("Error writing header to CSV:", err)
			return
		}
	}


	for _, element := range data {
		controllerKindMap := element.(map[string]interface{})

		for _, controllerKindData := range controllerKindMap {
			controllerKindOne := controllerKindData.(map[string]interface{})

			name := controllerKindOne["name"].(string)


			properties := controllerKindOne["properties"].(map[string]interface{})

			var region string
			if name != "__idle__" {
				labels := properties["labels"].(map[string]interface{})
				region = labels["topology_kubernetes_io_region"].(string)
			}

			window := controllerKindOne["window"].(map[string]interface{})
			windowStart := window["start"].(string)
			windowEnd := window["end"].(string)

			cpuCost := controllerKindOne["cpuCost"].(float64)
			gpuCost := controllerKindOne["gpuCost"].(float64)
			ramCost := controllerKindOne["ramCost"].(float64)
			pvCost := controllerKindOne["pvCost"].(float64)
			networkCost := controllerKindOne["networkCost"].(float64)
			loadBalancerCost := controllerKindOne["loadBalancerCost"].(float64)
			sharedCost := controllerKindOne["sharedCost"].(float64)
			totalCost := controllerKindOne["totalCost"].(float64)
			cpuEfficiency := controllerKindOne["cpuEfficiency"].(float64) * 100
			ramEfficiency := controllerKindOne["ramEfficiency"].(float64) * 100
			totalEfficiency := controllerKindOne["totalEfficiency"].(float64) * 100

			record := []string{
				name,clusterName,region, windowStart, windowEnd,
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

	configs.InfoLogger.Println("ControllerKind data successfully written to S3")
	wg.Done()
}
