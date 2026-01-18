package handlers

import (
	"context"
	"os"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "go-backend/pkg/config"
)

const (
	b2Endpoint = "https://s3.us-east-005.backblazeb2.com"
)

// getBucketName returns the S3 bucket name from global config
func (s *Handler) getBucketName() string {
	appCfg := appconfig.GetConfig()
	if appCfg == nil {
		log.Fatal("Configuration not loaded")
	}
	return appCfg.Services.S3.BucketName
}

func (s *Handler) CreateS3Client() *s3.Client {
	appCfg := appconfig.GetConfig()
	if appCfg == nil {
		log.Fatal("Configuration not loaded")
	}

	accessKeyID := appCfg.Services.S3.AccessKeyID
	secretAccessKey := appCfg.Services.S3.SecretAccessKey

	if accessKeyID == "" || secretAccessKey == "" {
		log.Fatal("B2_ACCESS_KEY_ID and B2_SECRET_ACCESS_KEY must be set")
	}

	// Create an AWS Config with the B2 credentials and S3 endpoint
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-005"), // Replace with your region
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		config.WithEndpointResolver(aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			if service == s3.ServiceID {
				return aws.Endpoint{
					URL: b2Endpoint,
				}, nil
			}
			return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
		})),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg)
	return client
}

func (s *Handler) listObjects(client *s3.Client) {
	// List the objects in the bucket
	resp, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(s.getBucketName()),
	})
	if err != nil {
		log.Fatalf("unable to list items in bucket %q, %v", s.getBucketName(), err)
	}

	for _, item := range resp.Contents {
		fmt.Printf("Name: %s, Size: %d\n", *item.Key, item.Size)
	}
}
func (s *Handler) uploadObject(client *s3.Client, key, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("unable to open file %q, %v", filePath, err)
	}
	defer file.Close()

	if s.Server.Testing {
		s.Server.TestInspector.FilesUploaded += 1
		return
	}
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.getBucketName()),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		log.Fatalf("unable to upload %q to %q, %v", filePath, s.getBucketName(), err)
	}
}

func (s *Handler) downloadObject(client *s3.Client, key, filePath string) (*s3.GetObjectOutput, error) {

	if s.Server.Testing {
		return nil, nil
	}
	result, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.getBucketName()),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to receive file")
	}
	return result, err

	// file, err := os.Create(filePath)
	// if err != nil {
	// 	log.Fatalf("unable to create file %q, %v", filePath, err)
	// }
	// defer file.Close()

	// _, err = io.Copy(file, resp.Body)
	// if err != nil {
	// 	log.Fatalf("unable to copy data to file %q, %v", filePath, err)
	// }

	// fmt.Printf("Successfully downloaded %q to %q\n", key, filePath)
}
func (s *Handler) deleteObject(client *s3.Client, key string) error {
	if s.Server.Testing {
		s.Server.TestInspector.FilesUploaded -= 1
		return nil
	}
	_, err := client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.getBucketName()),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Fatalf("unable to delete item %q, %v", key, err)
		return err
	}
	//	fmt.Printf("Successfully deleted %q from %q\n", key, s.getBucketName())
	return nil
}
