package repository

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoRepository encapsulates the Amazon DynamoDB service actions used in the examples.
// It contains a DynamoDB service client that is used to act on the specified table.
type DynamoRepository struct {
	DynamoDbClient *dynamodb.Client
	TableName      string
}

func New() *DynamoRepository {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	return &DynamoRepository{
		DynamoDbClient: dynamodb.NewFromConfig(sdkConfig),
		TableName:      os.Getenv("SERVER_TABLE_NAME"),
	}
}

func (d *DynamoRepository) WriteItem(ctx context.Context, key, value string) error {
	_, err := d.DynamoDbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.TableName),
		Item: map[string]types.AttributeValue{
			"id":       &types.AttributeValueMemberN{Value: key},
			"greeting": &types.AttributeValueMemberS{Value: value},
		},
	})
	return err
}

func (d *DynamoRepository) ReadItem(ctx context.Context, key string) (string, error) {
	out, err := d.DynamoDbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.TableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberN{Value: key},
		},
	})
	if err != nil {
		return "", err
	}
	result, ok := out.Item["greeting"].(*types.AttributeValueMemberS)
	if !ok {
		err := fmt.Errorf("Received unexpected type for greeting attribute, %T", out.Item["greeting"])
		return "", err
	}
	return result.Value, nil
}
