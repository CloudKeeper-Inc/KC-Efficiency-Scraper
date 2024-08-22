package configs

import (
	"log"
	"os"
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)
const (
	KubecostEndpoint = "<kubecost-dashboard-url>" // Example - http://xxxxxxxxxxx.ap-south-1.elb.amazonaws.com:9090
	ClusterName = "<Cluster-Name>"
	BucketName = "<bucket-name>"
	BucketRegion = "<bucket-region>"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger

	enddate = time.Now().Format("2006-01-02")
	startdate = time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	start = startdate + "T00:00:00Z"
	end = enddate + "T00:00:00Z"
	Window = start + "," + end    // Window represents the time range of yesterday. (Format - 2024-07-27T00:00:00Z,2024-07-28T00:00:00Z)

	Svc *s3.S3
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(BucketRegion),
	})
	if err != nil {
		ErrorLogger.Println("Error creating AWS session:", err)
		return
	}

	Svc = s3.New(sess)
}