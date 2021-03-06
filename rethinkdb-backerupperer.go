package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/davidbanham/required_env"
	"github.com/robfig/cron"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	required_env.Ensure(map[string]string{
		"AWS_ACCESS_KEY_ID":     "",
		"AWS_SECRET_ACCESS_KEY": "",
		"RETHINK_LOC":           "",
		"S3_BUCKET":             "",
		"AWS_REGION":            "us-east-1",
	})

	if os.Getenv("CRON_STRING") != "" {
		c := cron.New()
		err := c.AddFunc(os.Getenv("CRON_STRING"), doBackup)
		if err != nil {
			log.Fatal(err)
		}
		c.Start()
		select {}
	} else {
		doBackup()
	}
}

func doBackup() {
	filename := time.Now().Format(time.RFC3339) + ".tar.gz"
	cmd := exec.Command("rethinkdb", "dump", "-c", os.Getenv("RETHINK_LOC"), "-f", filename)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	config := aws.NewConfig().WithRegion(os.Getenv("AWS_REGION"))
	sess := session.New(config)
	svc := s3.New(sess)

	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}

	params := &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    &filename,
		Body:   file,
	}

	if os.Getenv("SSE_KEY") != "" {
		params.SSECustomerKey = aws.String(os.Getenv("SSE_KEY"))
		params.SSECustomerAlgorithm = aws.String("AES256")
	}

	_, err = svc.PutObject(params)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully uploaded backup " + filename)

	err = os.Remove(filename)
	if err != nil {
		log.Fatal(err)
	}
}
