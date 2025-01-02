package entity

type Config struct {
	AwsRegion  string `json:"aws_region"`
	BucketTest string `json:"bucket_test"`
	QueueTest  string `json:"queue_test"`
}
