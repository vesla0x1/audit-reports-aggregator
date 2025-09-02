package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"shared/application/ports"
	"shared/infrastructure/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSQueue struct {
	client  *sqs.Client
	logger  ports.Logger
	metrics ports.Metrics
	config  *config.SQSConfig
	// Cache queue URLs to avoid repeated lookups
	queueURLs map[string]string
}

func NewSQSQueue(cfg *config.SQSConfig, obs ports.Observability) (ports.Queue, error) {
	logger, metrics, err := obs.ComponentsScoped("queue.sqs")
	if err != nil {
		return nil, fmt.Errorf("failed to get observability components: %w", err)
	}

	// Create AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		logger.Error("failed to load AWS config", "error", err)
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SQS client
	client := sqs.NewFromConfig(awsCfg)

	logger.Info("SQS queue initialized successfully", "region", cfg.Region)

	return &SQSQueue{
		client:    client,
		logger:    logger,
		metrics:   metrics,
		config:    cfg,
		queueURLs: make(map[string]string),
	}, nil
}

func (q *SQSQueue) getQueueURL(ctx context.Context, queueName string) (string, error) {
	// Check cache
	if url, ok := q.queueURLs[queueName]; ok {
		return url, nil
	}

	// Get queue URL from AWS
	result, err := q.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get queue URL for %s: %w", queueName, err)
	}

	// Cache the URL
	q.queueURLs[queueName] = *result.QueueUrl
	return *result.QueueUrl, nil
}

func (q *SQSQueue) Publish(ctx context.Context, message *ports.QueueMessage) error {
	startTime := time.Now()
	defer func() {
		q.metrics.RecordHistogram("queue.publish.duration",
			time.Since(startTime).Seconds(),
			map[string]string{"target": message.Target})
	}()

	// Get queue URL
	queueURL, err := q.getQueueURL(ctx, message.Target)
	if err != nil {
		q.logger.Error("failed to get queue URL", "error", err, "queue", message.Target)
		q.metrics.IncrementCounter("queue.publish.error",
			map[string]string{"target": message.Target, "error": "queue_url_failed"})
		return err
	}

	// Marshal message body to JSON
	body, err := json.Marshal(message.Body)
	if err != nil {
		q.logger.Error("failed to marshal message", "error", err)
		q.metrics.IncrementCounter("queue.publish.error",
			map[string]string{"target": message.Target, "error": "marshal_failed"})
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Build SQS message
	sqsMsg := &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(body)),
	}

	// Send message
	_, err = q.client.SendMessage(ctx, sqsMsg)
	if err != nil {
		q.logger.Error("failed to send message", "error", err, "target", message.Target)
		q.metrics.IncrementCounter("queue.publish.error",
			map[string]string{"target": message.Target, "error": "send_failed"})
		return fmt.Errorf("failed to send message: %w", err)
	}

	q.logger.Info("message sent successfully", "target", message.Target, "size", len(body))
	q.metrics.IncrementCounter("queue.publish.success",
		map[string]string{"target": message.Target})

	return nil
}

func (q *SQSQueue) PublishBatch(ctx context.Context, messages []*ports.QueueMessage) error {
	// Group messages by target queue
	batches := make(map[string][]*ports.QueueMessage)
	for _, msg := range messages {
		batches[msg.Target] = append(batches[msg.Target], msg)
	}

	// Process each batch
	for target, batch := range batches {
		if err := q.publishBatchToQueue(ctx, target, batch); err != nil {
			return err
		}
	}

	return nil
}

func (q *SQSQueue) publishBatchToQueue(ctx context.Context, target string, messages []*ports.QueueMessage) error {
	// SQS has a limit of 10 messages per batch
	const maxBatchSize = 10

	for i := 0; i < len(messages); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(messages) {
			end = len(messages)
		}

		batch := messages[i:end]
		entries := make([]types.SendMessageBatchRequestEntry, len(batch))

		for j, msg := range batch {
			body, err := json.Marshal(msg.Body)
			if err != nil {
				return fmt.Errorf("failed to marshal message: %w", err)
			}

			entries[j] = types.SendMessageBatchRequestEntry{
				Id:          aws.String(fmt.Sprintf("%d", j)),
				MessageBody: aws.String(string(body)),
			}
		}

		queueURL, err := q.getQueueURL(ctx, target)
		if err != nil {
			return err
		}

		_, err = q.client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
			QueueUrl: aws.String(queueURL),
			Entries:  entries,
		})

		if err != nil {
			return fmt.Errorf("failed to send batch: %w", err)
		}
	}

	return nil
}
