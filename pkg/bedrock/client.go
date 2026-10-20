package bedrock

import (
	"context"
	"fmt"
	"strings"

	"a2ademo/pkg/config"
	"a2ademo/pkg/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

// LLMClient defines the interface for calling an LLM inference service.
type LLMClient interface {
	Converse(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
	ModelID() string
}

// Client wraps the AWS Bedrock Runtime client.
type Client struct {
	client  *bedrockruntime.Client
	modelID string
}

// NewClient creates an LLMClient backed by AWS Bedrock Runtime.
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWSRegion),
	}

	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			cfg.AWSSessionToken,
		)
		opts = append(opts, awsconfig.WithCredentialsProvider(creds))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	brClient := bedrockruntime.NewFromConfig(awsCfg)
	return &Client{
		client:  brClient,
		modelID: cfg.BedrockModelID,
	}, nil
}

// ModelID returns the configured model identifier.
func (c *Client) ModelID() string {
	return c.modelID
}

// Converse invokes the Bedrock Converse API with system and user prompts.
func (c *Client) Converse(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	logger.Log(logger.TagBedrock, "Invoking AWS Bedrock model: %s", c.modelID)

	input := &bedrockruntime.ConverseInput{
		ModelId: aws.String(c.modelID),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{
						Value: userPrompt,
					},
				},
			},
		},
	}

	if systemPrompt != "" {
		input.System = []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{
				Value: systemPrompt,
			},
		}
	}

	out, err := c.client.Converse(ctx, input)
	if err != nil {
		return "", fmt.Errorf("bedrock converse failed: %w", err)
	}

	var sb strings.Builder
	if msgOutput, ok := out.Output.(*types.ConverseOutputMemberMessage); ok {
		for _, block := range msgOutput.Value.Content {
			if textBlock, ok := block.(*types.ContentBlockMemberText); ok {
				sb.WriteString(textBlock.Value)
			}
		}
	}

	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("empty response received from Bedrock")
	}

	return result, nil
}
