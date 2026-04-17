package db

import (
	"context"
	"fmt"
)

type CreateProductTxParams struct {
	Code            string            `json:"code"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Price           int64             `json:"price"`
	Discount        int64             `json:"discount"`
	AttributeValues []AttributesValue `json:"attribute_values"`
}

type CreateProductTxResult struct {
	Product           Product             `json:"product"`
	ProductAttributes []ProductsAttribute `json:"product_attributes"`
}

func (store *SQLStore) CreateProductTx(ctx context.Context, arg CreateProductTxParams) (CreateProductTxResult, error) {
	var result CreateProductTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		createProductArg := CreateProductParams{
			Code:        arg.Code,
			Name:        arg.Name,
			Description: arg.Description,
			Price:       arg.Price,
			Discount:    arg.Discount,
		}
		fmt.Println(arg.Code)

		product, err := q.CreateProduct(ctx, createProductArg)
		if err != nil {
			return err
		}

		productAttributes := []ProductsAttribute{}

		for _, a := range arg.AttributeValues {
			addAttributeArg := CreateProductAttributeParams{
				ProductID: product.ID,
				Attribute: a.Attribute,
				// TODO: refactor to only use value
				AttributeValueID: a.ID,
			}
			productAttribute, err := q.CreateProductAttribute(ctx, addAttributeArg)
			if err != nil {
				return err
			}
			productAttributes = append(productAttributes, productAttribute)
		}

		result.Product = product
		result.ProductAttributes = productAttributes

		return nil
	})

	return result, err
}
