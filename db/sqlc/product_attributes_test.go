package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func createRandomProductAttribute(t *testing.T) ProductsAttribute {
	product := createRandomProduct(t)
	attribute_value := createRandomAttributeValue(t)

	arg := CreateProductAttributeParams{
		ProductID:        product.ID,
		AttributeValueID: attribute_value.ID,
		AttributeID:      attribute_value.AttributeID,
	}

	product_attribute, err := testQueries.CreateProductAttribute(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, product.ID, product_attribute.ProductID)
	require.Equal(t, attribute_value.AttributeID, product_attribute.AttributeID)
	require.Equal(t, attribute_value.ID, product_attribute.AttributeValueID)

	return product_attribute
}

func TestCreateProductAttribute(t *testing.T) {
	createRandomProductAttribute(t)
}

func TestGetProductAttribute(t *testing.T) {
	product_attribute := createRandomProductAttribute(t)

	arg := GetProductAttributeValueParams{
		ProductID:   product_attribute.ProductID,
		AttributeID: product_attribute.AttributeID,
	}

	product_attribute2, err := testQueries.GetProductAttributeValue(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, product_attribute.ProductID, product_attribute2.ProductID)
	require.Equal(t, product_attribute.AttributeID, product_attribute2.AttributeID)
}
