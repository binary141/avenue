package email

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"avenue/backend/shared"
)

// SESSender sends emails via AWS Simple Email Service (SES v2).
type SESSender struct {
	client *sesv2.Client
	from   string
}

// NewSESSender creates a SESSender from environment variables:
//
//	SES_FROM       - the From address (required)
//	AWS_REGION     - AWS region (required, or set via standard AWS env/config)
//	AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY - credentials (or use IAM role)
func NewSESSender() (*SESSender, error) {
	from := shared.GetEnv("SES_FROM", "")
	if from == "" {
		return nil, fmt.Errorf("email: SES_FROM is not set")
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("email: failed to load AWS config: %w", err)
	}

	return &SESSender{
		client: sesv2.NewFromConfig(cfg),
		from:   from,
	}, nil
}

// Send delivers the message via AWS SES.
func (s *SESSender) Send(msg Message) error {
	body := &types.Body{}

	if msg.HTML != "" {
		body.Html = &types.Content{Data: aws.String(msg.HTML)}
	}
	if msg.Text != "" {
		body.Text = &types.Content{Data: aws.String(msg.Text)}
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.from),
		Destination: &types.Destination{
			ToAddresses: []string{msg.To},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(msg.Subject)},
				Body:    body,
			},
		},
	}

	_, err := s.client.SendEmail(context.Background(), input)
	return err
}
