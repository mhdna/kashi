package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func createRandomProductAttributes(t *testing.T) []ProductsAttribute {
	product := createRandomProduct(t)
	attributeValues := createRandomAttributeValues(t)
	productAttributes := []ProductsAttribute{}

	for _, a := range attributeValues {
		addAttributeArg := CreateProductAttributeParams{
			ProductID:        product.ID,
			Attribute:        a.Attribute,
			AttributeValueID: a.ID,
		}
		productAttribute, err := testQueries.CreateProductAttribute(context.Background(), addAttributeArg)
		require.NoError(t, err)

		productAttributes = append(productAttributes, productAttribute)
	}

	return productAttributes
}

func TestCreateProductAttribute(t *testing.T) {
	createRandomProductAttributes(t)
}

func TestGetProductAttribute(t *testing.T) {
	productAttributes := createRandomProductAttributes(t)

	for _, productAttribute := range productAttributes {
		arg := GetProductAttributeValueParams{
			ProductID: productAttribute.ProductID,
			Attribute: productAttribute.Attribute,
		}

		productAttribute2, err := testQueries.GetProductAttributeValue(context.Background(), arg)
		require.NoError(t, err)
		require.Equal(t, productAttribute.ProductID, productAttribute2.ProductID)
		require.Equal(t, productAttribute.Attribute, productAttribute2.Attribute)
		require.Equal(t, productAttribute.AttributeValueID, productAttribute2.AttributeValueID)
	}
}

func TestListProductAttributes(t *testing.T) {
	for range 10 {
		createRandomProductAttributes(t)
	}

	limit := 5
	offset := 5

	arg := ListProductAttributesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	productAttributes, err := testQueries.ListProductAttributes(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, productAttributes, limit)
	for _, productAttribute := range productAttributes {
		// TODO: complete this
		require.NotEmpty(t, productAttribute)
	}
}

func TestUpdateProductAttribute(t *testing.T) {
	productAttributes := createRandomProductAttributes(t)
	require.NotEmpty(t, productAttributes)

	newAttributeValues := createRandomAttributeValues(t)
	require.NotEmpty(t, newAttributeValues)

	for i, productAttribute := range productAttributes {
		newValue := newAttributeValues[i%len(newAttributeValues)]
		arg := UpdateProductAttributeParams{
			ProductID:        productAttribute.ProductID,
			Attribute:        productAttribute.Attribute,
			AttributeValueID: newValue.ID,
		}
		err := testQueries.UpdateProductAttribute(context.Background(), arg)
		require.NoError(t, err)

		arg2 := GetProductAttributeValueParams{
			ProductID: productAttribute.ProductID,
			Attribute: productAttribute.Attribute,
		}
		productAttribute2, err := testQueries.GetProductAttributeValue(context.Background(), arg2)
		require.NoError(t, err)
		require.Equal(t, productAttribute2.ProductID, arg.ProductID)
		require.Equal(t, productAttribute2.Attribute, arg.Attribute)
		require.Equal(t, productAttribute2.AttributeValueID, arg.AttributeValueID)
	}
}

func TestGetProductAttributes(t *testing.T) {
	productAttributes := createRandomProductAttributes(t)
	require.NotEmpty(t, productAttributes)

	pID := productAttributes[0].ProductID

	fetched, err := testQueries.GetProductAttributes(context.Background(), pID)
	require.NoError(t, err)
	require.Len(t, fetched, len(productAttributes))

	for i, fa := range fetched {
		require.Equal(t, pID, fa.ProductID)
		require.Equal(t, productAttributes[i].Attribute, fa.Attribute)
		require.Equal(t, productAttributes[i].AttributeValueID, fa.AttributeValueID)
	}
}
