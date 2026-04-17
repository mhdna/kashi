package db

import (
	"context"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func createRandomProductAttribute(t *testing.T) ProductsAttribute {
	product := createRandomProduct(t)
	attribute_value := createRandomAttributeValue(t)

	arg := CreateProductAttributeParams{
		ProductID:        product.ID,
		AttributeValueID: attribute_value.ID,
		Attribute:        attribute_value.Attribute,
	}

	product_attribute, err := testQueries.CreateProductAttribute(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, product.ID, product_attribute.ProductID)
	require.Equal(t, attribute_value.Attribute, product_attribute.Attribute)
	require.Equal(t, attribute_value.ID, product_attribute.AttributeValueID)

	return product_attribute
}

func TestCreateProductAttribute(t *testing.T) {
	createRandomProductAttribute(t)
}

func TestGetProductAttribute(t *testing.T) {
	product_attribute := createRandomProductAttribute(t)

	arg := GetProductAttributeValueParams{
		ProductID: product_attribute.ProductID,
		Attribute: product_attribute.Attribute,
	}

	product_attribute2, err := testQueries.GetProductAttributeValue(context.Background(), arg)
	require.NoError(t, err)
	require.Equal(t, product_attribute.ProductID, product_attribute2.ProductID)
	require.Equal(t, product_attribute.Attribute, product_attribute2.Attribute)
}

func TestListProductAttributes(t *testing.T) {
	for range 10 {
		createRandomProductAttribute(t)
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

func TestUpdateProductAttributes(t *testing.T) {
	productAttribute := createRandomProductAttribute(t)

	attributesLength := int64(5)
	for range attributesLength {
		createRandomAttributeValue(t)
	}

	arg := UpdateProductAttributeParams{
		ProductID:        productAttribute.ProductID,
		Attribute:        productAttribute.Attribute,
		AttributeValueID: util.RandomInt(1, attributesLength),
	}
	err := testQueries.UpdateProductAttribute(context.Background(), arg)
	require.NoError(t, err)

	arg2 := GetProductAttributeValueParams{
		ProductID: productAttribute.ProductID,
		Attribute: productAttribute.Attribute,
	}
	productAttribute2, err := testQueries.GetProductAttributeValue(context.Background(), arg2)
	require.Equal(t, productAttribute2.ProductID, arg.ProductID)
	require.Equal(t, productAttribute2.Attribute, arg.Attribute)
	require.Equal(t, productAttribute2.AttributeValueID, arg.AttributeValueID)
}
