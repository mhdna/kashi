package db

import (
	"context"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomAttributeValues(t *testing.T) []AttributesValue {
	attributes, err := testQueries.ListAttributes(context.Background())
	require.NoError(t, err)

	attributeValues := make([]AttributesValue, 0, len(attributes))

	for _, a := range attributes {
		arg := CreateAttributeValueParams{
			Attribute: a,
			Value:     util.RandomString(6),
		}
		attributeValue, err := testQueries.CreateAttributeValue(context.Background(), arg)
		require.NoError(t, err)
		require.NotEmpty(t, attributeValue)
		require.Equal(t, attributeValue.Attribute, arg.Attribute)
		require.Equal(t, attributeValue.Value, arg.Value)

		attributeValues = append(attributeValues, attributeValue)
	}

	return attributeValues
}

func TestCreateAttribute(t *testing.T) {
	createRandomAttributeValues(t)
}

func TestGetAttributeValue(t *testing.T) {
	attributeValues := createRandomAttributeValues(t)

	for _, a := range attributeValues {
		attributeValue, err := testQueries.GetAttributeValue(context.Background(), a.ID)

		require.NoError(t, err)
		require.Equal(t, attributeValue.ID, a.ID)
		require.Equal(t, attributeValue.Attribute, a.Attribute)
		require.Equal(t, attributeValue.Value, a.Value)
	}
}

func TestListAttributeValues(t *testing.T) {
	for range 10 {
		createRandomAttributeValues(t)
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
	attributeValues := createRandomAttributeValues(t)

	for _, a := range attributeValues {
		arg := UpdateAttributeValueParams{
			ID:    a.ID,
			Value: util.RandomAttributeValue(),
		}

		attributeValue2, err := testQueries.UpdateAttributeValue(context.Background(), arg)
		require.NoError(t, err)
		require.Equal(t, attributeValue2.ID, arg.ID)
		require.Equal(t, attributeValue2.Value, arg.Value)
	}
}
