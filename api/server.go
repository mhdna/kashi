package api

import (
	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
)

type Server struct {
	store  db.Store
	router *gin.Engine
}

func NewServer(store db.Store) *Server {
	server := &Server{store: store}

	router := gin.Default()
	router.POST("/inventories", server.createInventory)
	router.GET("/inventories/:id", server.getInventory)
	router.GET("/inventories/", server.listInventories)
	router.POST("/products", server.createProduct)
	router.GET("/products/:id", server.getProduct)
	router.GET("/products/", server.listProducts)
	router.DELETE("/products/:id", server.deleteProduct)
	router.POST("/attributes", server.createAttributeValue)
	router.PUT("/attributes", server.updateAttributeValue)
	router.GET("/attributes/", server.listAttributeValues)
	// TODO: add updateAsset
	router.POST("/assets", server.createAsset)
	router.DELETE("/assets/:id", server.deleteAsset)
	router.GET("/assets/:id", server.getAsset)
	router.GET("/assets/", server.listAssets)
	// TODO: add getAssetType
	router.POST("/asset_types", server.createAssetType)
	router.DELETE("/asset_types/:id", server.deleteAssetType)
	router.POST("/cashboxes", server.createCashbox)
	router.GET("/cashboxes/:id", server.getCashbox)
	router.GET("/cashboxes/", server.listCashboxes)
	router.POST("/clients", server.createClient)
	router.PUT("/clients", server.updateClient)
	router.GET("/clients/:id", server.getClient)
	router.GET("/clients/", server.listClients)
	router.POST("/currencies", server.createCurrency)
	router.GET("/currencies/:id", server.getCurrency)
	router.GET("/currencies/", server.listCurrencies)
	router.POST("/shifts", server.createShift)
	router.POST("/shifts/:id/close", server.CloseShift)
	router.GET("/shifts", server.listShifts)
	router.GET("/shifts/:id", server.getShift)

	router.POST("/suppliers", server.createSupplier)
	router.GET("/suppliers/:id", server.getSupplier)
	router.GET("/suppliers", server.listSuppliers)

	server.router = router
	return server
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) *gin.H {
	return &gin.H{"error": err.Error()}
}
