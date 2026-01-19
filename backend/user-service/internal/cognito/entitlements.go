package cognito

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// UpdateUserEntitlement updates the custom:entitlements attribute for a user
func UpdateUserEntitlement(ctx context.Context, client API, userPoolID, username, entitlement string) error {
	_, err := client.AdminUpdateUserAttributes(ctx, &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(username),
		UserAttributes: []types.AttributeType{
			{
				Name:  aws.String("custom:entitlements"),
				Value: aws.String(entitlement),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("update user entitlements: %w", err)
	}

	return nil
}

// GetUserEntitlement retrieves the current entitlement for a user
func GetUserEntitlement(ctx context.Context, client API, userPoolID, username string) (string, error) {
	resp, err := client.AdminGetUser(ctx, &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(username),
	})
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}

	for _, attr := range resp.UserAttributes {
		if aws.ToString(attr.Name) == "custom:entitlements" {
			return aws.ToString(attr.Value), nil
		}
	}

	return "free", nil // Default to free if not set
}
