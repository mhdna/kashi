package db

import (
	"context"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomAttributeValue(t *testing.T) AttributesValue {
	arg := CreateAttributeValueParams{
		AttributeID: util.RandomInt(1, 9),
		Value:       util.RandomString(6),
	}

	attribute, err := testQueries.CreateAttributeValue(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, attribute)
	require.Equal(t, attribute.AttributeID, arg.AttributeID)
	require.Equal(t, attribute.Value, arg.Value)

	return attribute
}

func TestGetAttributeValue(t *testing.T) {
	attribute_value := createRandomAttributeValue(t)

	attribute_value2, err := testQueries.GetAttributeValue(context.Background(), attribute_value.ID)

	require.NoError(t, err)
	require.Equal(t, attribute_value.ID, attribute_value2.ID)
	require.Equal(t, attribute_value.AttributeID, attribute_value2.AttributeID)
	require.Equal(t, attribute_value.Value, attribute_value2.Value)
}

func TestCreateAttribute(t *testing.T) {
	createRandomAttributeValue(t)
}
