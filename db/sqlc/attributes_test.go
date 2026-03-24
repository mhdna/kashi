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

func TestListAttributeValues(t *testing.T) {
	for range 10 {
		createRandomAttributeValue(t)
	}

	limit := 5
	offset := 5

	arg := ListAttributeValuesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	attribute_values, err := testQueries.ListAttributeValues(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, attribute_values, limit)
	for _, attrattribute_value := range attribute_values {
		require.NotEmpty(t, attrattribute_value)
	}
}

func TestUpdateAttributeValue(t *testing.T) {
	attribute_value1 := createRandomAttributeValue(t)
	arg := UpdateAttributeValueParams{
		ID:    attribute_value1.ID,
		Value: util.RandomAttributeValue(),
	}

	err := testQueries.UpdateAttributeValue(context.Background(), arg)
	require.NoError(t, err)

	attribute_value2, err := testQueries.GetAttributeValue(context.Background(), attribute_value1.ID)
	require.Equal(t, attribute_value2.ID, arg.ID)
	require.Equal(t, attribute_value2.Value, arg.Value)
}
