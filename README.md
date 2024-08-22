# Kubecost Allocation API Fetcher

This Go script fetches data from the Kubecost Allocation API to retrieve efficiency metrics.

Written by -
[Akash Sawan at CloudKeeper](https://github.com/akashsawan1)

## Prerequisites

- Go (Golang) installed on your machine
- Kubecost Dashboard API endpoint

## Setup

1. Clone the repository to your local machine.
2. Navigate into the project directory.
3. Open the config.go file located in the configs package
4. Update the Kubecost Dashboard Endpoint:
    - Locate the line defining `KubecostEndpoint` in `config.go`.
    - Replace the placeholder text with your actual Kubecost dashboard endpoint.
5. Update the Cluster Name:
    - Locate the line defining `ClusterName` in `config.go`.
    - Replace the placeholder text with your Cluster name.
6. Update the S3 Bucket Name:
    - Locate the line defining `BucketName` in `config.go`.
    - Replace the placeholder text with your actual S3 bucket name.
7. Update the S3 Bucket Region:
    - Locate the line defining `BucketRegion` in `config.go`.
    - Replace the placeholder text with your actual S3 bucket region.

## Optional

If you want to get the data for specific dates, comment out the existing `Window` definition and define a new `Window` variable with the desired date range in the format `2024-07-20T00:00:00Z,2024-07-21T00:00:00Z`.

```sh
//e.g.
// Custom range
Window := "2024-07-27T00:00:00Z,2024-07-28T00:00:00Z"
```



## Running the Code


To run the script without building using go run:

Note : You should be inside the project directory.

```sh
go run .
```


## Building and Running the Executable

To build the executable:

Note : You should be inside the project directory.

```sh
go build .
```

Then, run the built executable:
```sh
./kubecost-efficiency-fetcher
```
