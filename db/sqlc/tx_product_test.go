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

	n := 5
	names := make([]string, n)
	codes := make([]string, n)
	descriptions := make([]string, n)
	discounts := make([]int16, n)
	attributeValuesArray := make([][]AttributesValue, n)

	for i := range n {
		names[i] = util.RandomString(20)
		codes[i] = util.RandomString(8)
		descriptions[i] = util.RandomString(200)
		discounts[i] = util.RandomDiscount()
		attributeValuesArray[i] = createRandomAttributeValues(t)
	}

	errs := make(chan error)

	type indexedResult struct {
		res CreateProductTxResult
		idx int
	}

	results := make(chan indexedResult)

	for i := range n {
		go func(idx int) {
			res, err := store.CreateProductTx(context.Background(), CreateProductTxParams{
				Code:            codes[idx],
				Name:            names[idx],
				Description:     descriptions[idx],
				Discount:        discounts[idx],
				AttributeValues: attributeValuesArray[idx],
			})

			errs <- err
			results <- indexedResult{idx: idx, res: res}
		}(i)
	}

	for range n {
		err := <-errs
		require.NoError(t, err)

		indexedRes := <-results
		res := indexedRes.res
		idx := indexedRes.idx

		resProduct := res.Product
		require.NotEmpty(t, resProduct)
		require.Equal(t, codes[idx], resProduct.Code)
		require.Equal(t, descriptions[idx], resProduct.Description)
		require.NotZero(t, resProduct.ID)
		require.NotZero(t, resProduct.CreatedAt)
		fmt.Println(">> tx product:", res.Product.Code)

		resProductAttributes := res.ProductAttributes
		for _, attributeValue := range attributeValuesArray[idx] {
			for _, resProductAttribute := range resProductAttributes {
				if resProductAttribute.AttributeID == attributeValue.AttributeID {
					fmt.Println(">> attribute:", resProductAttribute.AttributeID, attributeValue.AttributeID, resProductAttribute.AttributeValueID, attributeValue.ID)
					require.Equal(t, attributeValue.AttributeID, resProductAttribute.AttributeID)
					require.Equal(t, attributeValue.ID, resProductAttribute.AttributeValueID)
				}
			}
		}
	}
}
