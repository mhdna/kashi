package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/mhdna/kashi/util"
	"github.com/stretchr/testify/require"
)

func TestCreateProductTx(t *testing.T) {
	store := NewStore(testDB)
	name := util.RandomString(20)
	code := util.RandomString(8)
	description := util.RandomString(200)
	price := util.RandomNumber()
	discount := util.RandomNumber()
	attributeValues := createRandomAttributeValues(t)

	errs := make(chan error)
	results := make(chan CreateProductTxResult)

	go func() {
		res, err := store.CreateProductTx(context.Background(), CreateProductTxParams{
			Code:            code,
			Name:            name,
			Description:     description,
			Price:           price,
			Discount:        discount,
			AttributeValues: attributeValues,
		})
		errs <- err
		results <- res
	}()

	err := <-errs
	require.NoError(t, err)

	res := <-results

	resProduct := res.Product
	require.NotEmpty(t, resProduct)
	require.Equal(t, code, resProduct.Code)
	require.Equal(t, description, resProduct.Description)
	require.Equal(t, discount, resProduct.Discount)
	require.NotZero(t, resProduct.ID)
	require.NotZero(t, resProduct.CreatedAt)

	resProductAttributes := res.ProductAttributes
	for _, attributeValue := range attributeValues {
		for _, resProductAttribute := range resProductAttributes {
			if resProductAttribute.Attribute == attributeValue.Attribute {
				fmt.Println(">> tx product attribute:", resProductAttribute.Attribute, attributeValue.Attribute, resProductAttribute.AttributeValueID, attributeValue.ID)
				require.Equal(t, attributeValue.Attribute, resProductAttribute.Attribute)
				require.Equal(t, attributeValue.ID, resProductAttribute.AttributeValueID)
			}
		}
	}
}
