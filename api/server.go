package api

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	db "github.com/mhdna/kashi/db/sqlc"
	"github.com/mhdna/kashi/token"
	"github.com/mhdna/kashi/util"
)

type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
	branchHub  *branchHub
	adminHub   *adminHub
	// recordActivity writes the home page's "In kashi" feed
	// (admin_activity.go); tests replace it.
	recordActivity func(adminActivity)
	// security counts, locates and filters every request (security.go).
	security *securityGuard
}

func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %s", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		branchHub:  newBranchHub(),
		adminHub:   newAdminHub(),
	}
	server.security = newSecurityGuard(store, newGeoIP(config.GeoIPDir))
	server.security.notify = func() {
		server.adminHub.broadcastAll(adminWSMessage{Type: "security_alert"})
	}

	server.recordActivity = server.recordAdminActivity
	server.setupRoutes()

	return server, nil
}

// RunSecurity keeps the request guard's settings, bans and visitor counts
// in step with the database and the IP-location databases current, until
// ctx ends (security.go). main starts it; tests don't.
func (server *Server) RunSecurity(ctx context.Context) {
	server.security.Run(ctx)
}

// BranchHub exposes the server's live-push hub so background schedulers
// (e.g. RunBranchTargetSeriesScheduler) that run outside a request's
// lifecycle can notify connected branches too, using the same connections
// branch API handlers push through.
func (server *Server) BranchHub() *branchHub {
	return server.branchHub
}

func (server *Server) setupRoutes() {
	// Persist Gin access and recovery output through the same bounded writer
	// configured for the standard Go logger while keeping Gin's middleware.
	gin.DefaultWriter = log.Writer()
	gin.DefaultErrorWriter = log.Writer()
	router := gin.Default()
	// Caddy, on this same machine, is the only proxy in front: trust its
	// X-Forwarded-For and nobody else's, so a client can't claim another IP.
	if err := router.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		log.Printf("trusted proxies: %v", err)
	}
	router.Use(corsMiddleware(), server.security.middleware())

	authRoutes := router.Group("/").Use(server.authMiddleware(), server.permissionMiddleware(), server.adminActivityMiddleware())

	authRoutes.POST("/inventories", server.createInventory)
	authRoutes.GET("/inventories/:id", server.getInventory)
	authRoutes.GET("/inventories/", server.listInventories)
	authRoutes.PUT("/inventories", server.updateInventory)
	authRoutes.DELETE("/inventories/:id", server.deleteInventory)
	authRoutes.POST("/inventories/bulk_delete", server.bulkDeleteInventories)
	authRoutes.GET("/inventories/:id/stock", server.listInventoryStock)
	authRoutes.POST("/inventories/:id/adjustments", server.createStockAdjustment)
	authRoutes.GET("/stock_movements", server.listStockMovements)
	authRoutes.GET("/audit_logs", server.listAuditLogs)
	authRoutes.GET("/admin_activities", server.listAdminActivities)
	authRoutes.GET("/home/latest", server.listLatestAdditions)
	authRoutes.GET("/home/activities", server.listHomeActivities)
	authRoutes.GET("/home/activities/urgent", server.listUrgentHomeActivities)
	authRoutes.GET("/home/activities/detail", server.getHomeActivityDetail)
	authRoutes.POST("/home/activities/resolve", server.resolveHomeActivity)
	authRoutes.DELETE("/home/activities/resolve", server.unresolveHomeActivity)
	authRoutes.GET("/home/today", server.getHomeToday)
	authRoutes.GET("/low_stock", server.listLowStockProducts)
	authRoutes.GET("/low_stock/summary", server.getLowStockSummary)
	authRoutes.GET("/low_stock/ignored", server.listIgnoredLowStockProducts)
	authRoutes.GET("/stock_health", server.getStockHealth)
	authRoutes.GET("/stock_health/by_inventory", server.listStockHealthByInventory)
	authRoutes.GET("/stock_health/slow_movers", server.listSlowMovingProducts)
	authRoutes.PUT("/products/:id/low_stock_threshold", server.setProductLowStockThreshold)
	authRoutes.PUT("/products/:id/low_stock_alerts", server.setProductLowStockAlertsEnabled)
	authRoutes.PUT("/app_settings/default_low_stock_threshold", server.setDefaultLowStockThreshold)
	authRoutes.POST("/stock_counts", server.createStockCount)
	authRoutes.PUT("/stock_counts", server.updateStockCount)
	authRoutes.GET("/stock_counts/:id", server.getStockCount)
	authRoutes.GET("/stock_counts", server.listStockCounts)
	authRoutes.POST("/stock_counts/:id/post", server.postStockCount)
	authRoutes.POST("/stock_counts/:id/cancel", server.cancelStockCount)
	authRoutes.GET("/inventories/:id/count_sheet", server.listInventoryCountSheet)
	authRoutes.GET("/inventories/treemap", server.getWarehouseTreemap)
	authRoutes.POST("/products", server.createProduct)
	authRoutes.POST("/products/import", server.importProductsCSV)
	authRoutes.PUT("/products", server.updateProduct)
	authRoutes.GET("/products/:id", server.getProduct)
	authRoutes.GET("/products", server.listProducts)
	authRoutes.GET("/product_revenue", server.listProductRevenue)
	authRoutes.GET("/product_revenue/daily", server.listProductRevenueDaily)
	authRoutes.GET("/branch_product_revenue", server.listBranchProductRevenue)
	authRoutes.GET("/branch_product_revenue/daily", server.listBranchProductRevenueDaily)
	authRoutes.GET("/branch_product_revenue/by_branch", server.listBranchRevenueByBranch)
	authRoutes.PUT("/product_variants", server.updateProductVariant)
	authRoutes.POST("/product_variants", server.createProductVariant)
	authRoutes.DELETE("/product_variants/:id", server.deleteProductVariant)
	authRoutes.GET("/barcodes", server.listBarcodes)
	authRoutes.POST("/barcodes/assign", server.assignBarcodes)
	authRoutes.POST("/barcodes/print", server.printBarcodes)
	authRoutes.DELETE("/products/:id", server.deleteProduct)
	authRoutes.POST("/attributes/batch", server.createAttributeValues)
	authRoutes.POST("/attributes", server.createAttributeValue)
	authRoutes.PUT("/attributes", server.updateAttributeValue)
	authRoutes.GET("/attributes/", server.listAttributeValues)
	authRoutes.GET("/attribute_types", server.listAttributes)
	authRoutes.GET("/attribute_values", server.listAllAttributeValues)
	authRoutes.DELETE("/attributes/:id", server.deleteAttributeValue)
	authRoutes.POST("/attributes/bulk_delete", server.bulkDeleteAttributeValues)
	authRoutes.POST("/assets", server.createAsset)
	authRoutes.PUT("/assets", server.updateAsset)
	authRoutes.DELETE("/assets/:id", server.deleteAsset)
	authRoutes.POST("/assets/bulk_delete", server.bulkDeleteAssets)
	authRoutes.GET("/assets/treemap", server.getAssetsTreemap)
	authRoutes.GET("/assets/:id", server.getAsset)
	authRoutes.GET("/assets/", server.listAssets)
	authRoutes.POST("/asset_categories", server.createAssetCategory)
	authRoutes.GET("/asset_categories/:id", server.getAssetCategory)
	authRoutes.GET("/asset_categories", server.listAssetCategories)
	authRoutes.PUT("/asset_categories", server.updateAssetCategory)
	authRoutes.DELETE("/asset_categories/:id", server.deleteAssetCategory)
	authRoutes.POST("/product_categories", server.createProductCategory)
	authRoutes.GET("/product_categories/:id", server.getProductCategory)
	authRoutes.GET("/product_categories", server.listProductCategories)
	authRoutes.PUT("/product_categories", server.updateProductCategory)
	authRoutes.DELETE("/product_categories/:id", server.deleteProductCategory)
	authRoutes.POST("/clients", server.createClient)
	authRoutes.PUT("/clients", server.updateClient)
	authRoutes.GET("/clients/:id", server.getClient)
	authRoutes.GET("/clients/:id/loyalty_total", server.getClientLoyaltyTotal)
	authRoutes.GET("/clients/:id/profile", server.getClientProfile)
	authRoutes.GET("/clients/", server.listClients)
	authRoutes.GET("/clients/by_spending", server.listClientsBySpending)
	authRoutes.GET("/clients/spending_summary", server.listClientSpendingSummary)
	authRoutes.DELETE("/clients/:id", server.deleteClient)
	authRoutes.PUT("/clients/:id/active", server.setClientActive)
	authRoutes.POST("/currencies", server.createCurrency)
	authRoutes.GET("/currencies/:code", server.getCurrency)
	authRoutes.GET("/currencies/", server.listCurrencies)
	authRoutes.PUT("/currencies", server.updateCurrency)
	authRoutes.DELETE("/currencies/:code", server.deleteCurrency)
	authRoutes.POST("/currencies/bulk_delete", server.bulkDeleteCurrencies)

	authRoutes.POST("/cashboxes", server.createCashbox)
	authRoutes.GET("/cashboxes/:id", server.getCashbox)
	authRoutes.GET("/cashboxes/", server.listCashboxes)
	authRoutes.PUT("/cashboxes", server.updateCashbox)
	authRoutes.POST("/shifts", server.createShift)
	authRoutes.POST("/shifts/:id/close", server.CloseShift)
	authRoutes.GET("/shifts", server.listShifts)
	authRoutes.GET("/shifts/:id", server.getShift)
	authRoutes.POST("/cashbox_accounts", server.createCashboxAccount)
	authRoutes.GET("/cashbox_accounts/", server.listCashboxAccounts)
	authRoutes.PUT("/cashbox_accounts", server.updateCashboxAccount)
	authRoutes.PUT("/cashbox_accounts/:id/active", server.setCashboxAccountActive)
	authRoutes.DELETE("/cashbox_accounts/:id", server.deleteCashboxAccount)
	authRoutes.POST("/cashbox_accounts/balance", server.addCashboxAccountBalance)

	authRoutes.GET("/pos_settings", server.posSettings)

	authRoutes.GET("/app_settings", server.getAppSettings)
	authRoutes.PUT("/app_settings", server.updateAppSettings)
	authRoutes.PUT("/app_settings/product_autofill_rules", server.setProductAutofillRules)

	authRoutes.GET("/email_settings", server.getEmailSettings)
	authRoutes.PUT("/email_settings", server.updateEmailSettings)
	authRoutes.GET("/email_reports", server.listEmailReports)
	authRoutes.POST("/email_reports", server.sendEmailReport)
	authRoutes.GET("/email_reports/preview", server.previewEmailReport)
	authRoutes.GET("/email_reports/shifts", server.listEmailReportShifts)
	authRoutes.GET("/email_reports/recipients", server.listEmailReportRecipients)
	authRoutes.GET("/email_reports/registers", server.listEmailReportRegisters)
	authRoutes.GET("/email_schedules", server.listEmailSchedules)
	authRoutes.POST("/email_schedules", server.createEmailSchedule)
	authRoutes.PUT("/email_schedules/:id", server.updateEmailSchedule)
	authRoutes.DELETE("/email_schedules/:id", server.deleteEmailSchedule)
	authRoutes.POST("/email_schedules/:id/run", server.runEmailScheduleNow)
	authRoutes.GET("/whatsapp_settings", server.getWhatsAppSettings)
	authRoutes.PUT("/whatsapp_settings", server.updateWhatsAppSettings)
	authRoutes.GET("/whatsapp/rules", server.listWhatsAppRules)
	authRoutes.PUT("/whatsapp/rules", server.replaceWhatsAppRules)
	authRoutes.POST("/whatsapp/rules/test", server.testWhatsAppRules)
	authRoutes.PUT("/whatsapp/cost", server.setWhatsAppCost)
	authRoutes.GET("/whatsapp/stats", server.getWhatsAppStats)
	authRoutes.GET("/whatsapp/messages", server.listWhatsAppMessages)
	authRoutes.GET("/whatsapp/messages/:id/conversation", server.getWhatsAppConversation)
	authRoutes.GET("/whatsapp/poll_replies/:id/media", server.getWhatsAppReplyMedia)
	authRoutes.GET("/salesperson_performance", server.getSalespersonPerformance)
	authRoutes.GET("/whatsapp/clients", server.listWhatsAppClients)
	authRoutes.GET("/whatsapp/replies", server.listWhatsAppReplies)
	// Sending from our own numbers through the bridge (whatsapp_bridge.go).
	authRoutes.GET("/whatsapp/sender", server.getWhatsAppSender)
	authRoutes.PUT("/whatsapp/sender", server.updateWhatsAppSender)
	authRoutes.POST("/whatsapp/sender/test", server.sendWhatsAppTest)
	authRoutes.POST("/whatsapp/numbers", server.createWhatsAppNumber)
	authRoutes.GET("/whatsapp/numbers/:id/qr", server.whatsAppNumberQR)
	authRoutes.PUT("/whatsapp/numbers/:id", server.updateWhatsAppNumber)
	authRoutes.DELETE("/whatsapp/numbers/:id", server.deleteWhatsAppNumber)
	authRoutes.POST("/whatsapp/numbers/:id/resume", server.resumeWhatsAppNumber)
	authRoutes.GET("/whatsapp/health", server.getWhatsAppHealth)
	// The Reviews page (review_stats.go, review_link.go).
	authRoutes.GET("/reviews", server.getReviews)
	authRoutes.POST("/reviews/refresh", server.refreshReviews)
	authRoutes.GET("/reviews/settings", server.getReviewSettings)
	authRoutes.PUT("/reviews/settings", server.updateReviewSettings)

	authRoutes.GET("/calendar_events", server.listCalendarEvents)
	authRoutes.POST("/calendar_events", server.createCalendarEvent)
	authRoutes.DELETE("/calendar_events/:id", server.deleteCalendarEvent)
	authRoutes.GET("/logs", server.listAppLogs)

	authRoutes.POST("/suppliers", server.createSupplier)
	authRoutes.GET("/suppliers/:id", server.getSupplier)
	authRoutes.GET("/suppliers", server.listSuppliers)

	authRoutes.POST("/colors", server.createColor)
	authRoutes.GET("/colors", server.listColors)
	authRoutes.PUT("/colors", server.updateColor)
	authRoutes.DELETE("/colors/:id", server.deleteColor)
	authRoutes.POST("/colors/bulk_delete", server.bulkDeleteColors)
	authRoutes.POST("/sizes", server.createSize)
	authRoutes.GET("/sizes", server.listSizes)
	authRoutes.PUT("/sizes", server.updateSize)
	authRoutes.DELETE("/sizes/:id", server.deleteSize)
	authRoutes.POST("/sizes/bulk_delete", server.bulkDeleteSizes)

	authRoutes.POST("/sales_invoices", server.createSalesInvoice)
	authRoutes.GET("/sales_invoices/:id", server.getSalesInvoice)
	authRoutes.GET("/sales_invoices", server.listSalesInvoices)

	authRoutes.POST("/return_invoices", server.createReturnInvoice)
	authRoutes.GET("/return_invoices/:id", server.getReturnInvoice)
	authRoutes.GET("/return_invoices", server.listReturnInvoices)

	authRoutes.GET("/invoices/:id/details", server.getInvoiceDetails)
	authRoutes.GET("/branch_invoices/:id/details", server.getBranchInvoiceDetails)

	authRoutes.POST("/invoice_types", server.createInvoiceType)
	authRoutes.GET("/invoice_types/:id", server.getInvoiceType)
	authRoutes.GET("/invoice_types", server.listInvoiceTypes)
	authRoutes.PUT("/invoice_types", server.updateInvoiceType)

	authRoutes.POST("/price_lists", server.createPriceList)
	authRoutes.GET("/price_lists/:id", server.getPriceList)
	authRoutes.GET("/price_lists", server.listPriceLists)
	authRoutes.PUT("/price_lists", server.updatePriceList)
	authRoutes.DELETE("/price_lists/:id", server.deletePriceList)
	authRoutes.POST("/price_lists/items", server.createPriceListItem)
	authRoutes.GET("/price_lists/:id/items", server.listPriceListItems)
	authRoutes.DELETE("/price_lists/:id/items/:product_variant_id", server.deletePriceListItem)
	authRoutes.POST("/price_lists/:id/items/bulk_delete", server.bulkDeletePriceListItems)
	authRoutes.GET("/price_lists/:id/branches", server.listPriceListBranches)
	authRoutes.PUT("/price_lists/:id/branches", server.setPriceListBranches)

	authRoutes.POST("/discount_lists", server.createDiscountList)
	authRoutes.GET("/discount_lists/:id", server.getDiscountList)
	authRoutes.GET("/discount_lists", server.listDiscountLists)
	authRoutes.PUT("/discount_lists", server.updateDiscountList)
	authRoutes.DELETE("/discount_lists/:id", server.deleteDiscountList)
	authRoutes.POST("/discount_lists/items", server.createDiscountListItem)
	authRoutes.GET("/discount_lists/:id/items", server.listDiscountListItems)
	authRoutes.DELETE("/discount_lists/:id/items/:product_variant_id", server.deleteDiscountListItem)
	authRoutes.POST("/discount_lists/:id/items/bulk_delete", server.bulkDeleteDiscountListItems)
	authRoutes.GET("/discount_lists/:id/branches", server.listDiscountListBranches)
	authRoutes.PUT("/discount_lists/:id/branches", server.setDiscountListBranches)

	router.POST("/users/login", server.loginUser)
	// Meta's WhatsApp webhook: no login, every POST is signed (whatsapp_webhook.go).
	router.GET(whatsAppWebhookPath, server.verifyWhatsAppWebhook)
	router.POST(whatsAppWebhookPath, server.receiveWhatsAppWebhook)
	// Review links in the WhatsApp message (review_link.go): public, reached
	// through grandbz.com/review*.
	router.GET("/review", server.openReviewLink)
	router.GET("/review/:code", server.openReviewLink)
	router.GET(reviewBannerPath, server.reviewBanner)
	// Votes on the rating poll, from the WhatsApp bridge on this server
	// (whatsapp_poll.go); it signs with its API key.
	router.POST(whatsAppPollVotesPath, server.receiveWhatsAppPollVote)
	router.POST("/tokens/renew_access", server.renewAccessToken)
	authRoutes.POST("/users", server.createUser)
	authRoutes.GET("/users/online", server.listOnlineUsers)
	authRoutes.GET("/users/me", server.getMe)
	authRoutes.PUT("/users/me/avatar_color", server.setMyAvatarColor)
	authRoutes.GET("/users/:id", server.getUser)
	authRoutes.GET("/users", server.listUsers)
	authRoutes.PUT("/users", server.updateUser)
	authRoutes.DELETE("/users/:id", server.deleteUser)

	authRoutes.GET("/roles", server.listRoles)
	authRoutes.POST("/roles", server.createRole)
	authRoutes.PUT("/roles/:id", server.updateRole)
	authRoutes.DELETE("/roles/:id", server.deleteRole)

	authRoutes.POST("/entries", server.createEntry)
	authRoutes.GET("/entries/:id", server.getEntry)
	authRoutes.GET("/entries", server.listEntries)

	authRoutes.POST("/expense_categories", server.createExpenseCategory)
	authRoutes.GET("/expense_categories/:id", server.getExpenseCategory)
	authRoutes.GET("/expense_categories", server.listExpenseCategories)
	authRoutes.PUT("/expense_categories", server.updateExpenseCategory)
	authRoutes.DELETE("/expense_categories/:id", server.deleteExpenseCategory)

	authRoutes.POST("/expenses", server.createExpense)
	authRoutes.PUT("/expenses", server.updateExpense)
	authRoutes.DELETE("/expenses/:id", server.deleteExpense)
	authRoutes.POST("/expenses/bulk_delete", server.bulkDeleteExpenses)
	authRoutes.GET("/expenses/:id", server.getExpense)
	authRoutes.GET("/expenses", server.listExpenses)

	authRoutes.POST("/recurring_expenses", server.createRecurringExpense)
	authRoutes.GET("/recurring_expenses", server.listRecurringExpenses)
	authRoutes.PUT("/recurring_expenses/active", server.setRecurringExpenseActive)

	authRoutes.POST("/loan_categories", server.createLoanCategory)
	authRoutes.GET("/loan_categories/:id", server.getLoanCategory)
	authRoutes.GET("/loan_categories", server.listLoanCategories)
	authRoutes.PUT("/loan_categories", server.updateLoanCategory)
	authRoutes.DELETE("/loan_categories/:id", server.deleteLoanCategory)

	authRoutes.POST("/loans", server.createLoan)
	authRoutes.GET("/loans/:id", server.getLoan)
	authRoutes.GET("/loans", server.listLoans)
	authRoutes.PUT("/loans", server.updateLoan)
	authRoutes.DELETE("/loans/:id", server.deleteLoan)
	authRoutes.POST("/loans/bulk_delete", server.bulkDeleteLoans)

	authRoutes.POST("/loan_payments", server.createLoanPayment)
	authRoutes.GET("/loan_payments", server.listLoanPayments)
	authRoutes.DELETE("/loan_payments/:id", server.deleteLoanPayment)

	authRoutes.POST("/coupon_categories", server.createCouponCategory)
	authRoutes.GET("/coupon_categories/:id", server.getCouponCategory)
	authRoutes.GET("/coupon_categories", server.listCouponCategories)
	authRoutes.PUT("/coupon_categories", server.updateCouponCategory)
	authRoutes.DELETE("/coupon_categories/:id", server.deleteCouponCategory)

	authRoutes.POST("/coupon_types", server.createCouponType)
	authRoutes.GET("/coupon_types/:id", server.getCouponType)
	authRoutes.GET("/coupon_types", server.listCouponTypes)
	authRoutes.PUT("/coupon_types/:id", server.updateCouponType)
	authRoutes.DELETE("/coupon_types/:id", server.deleteCouponType)

	authRoutes.POST("/coupon_lists", server.createCouponList)
	authRoutes.GET("/coupon_lists", server.listCouponLists)
	authRoutes.POST("/coupon_lists/:id/deactivate", server.deactivateCouponList)

	authRoutes.POST("/coupons", server.createCoupon)
	authRoutes.POST("/coupons/bulk_delete", server.bulkDeleteCoupons)
	authRoutes.GET("/coupons/:code", server.getCoupon)
	authRoutes.GET("/coupons", server.listCoupons)
	authRoutes.PUT("/coupons/:code", server.updateCoupon)
	authRoutes.PUT("/coupons/:code/deactivate", server.deactivateCoupon)

	authRoutes.POST("/transfers", server.createTransfer)
	authRoutes.GET("/transfers/:id", server.getTransfer)
	authRoutes.GET("/transfers", server.listTransfers)
	authRoutes.PUT("/transfers", server.updateTransfer)
	authRoutes.POST("/transfers/:id/dispatch", server.dispatchTransfer)
	authRoutes.POST("/transfers/:id/receive", server.receiveTransfer)
	authRoutes.POST("/transfers/:id/approve", server.approveTransfer)
	authRoutes.POST("/transfers/:id/cancel", server.cancelTransfer)
	authRoutes.GET("/transfers/:id/attachment", server.getTransferAttachment)
	authRoutes.PUT("/transfers/:id/attachment/rotation", server.rotateTransferAttachment)
	authRoutes.PUT("/transfers/:id/attachment/replace", server.replaceTransferAttachment)
	authRoutes.GET("/transfer_attachments", server.listTransferAttachments)
	authRoutes.POST("/transfer_items", server.createTransferItem)
	authRoutes.GET("/transfer_items/:transfer_id", server.listTransferItems)
	authRoutes.PUT("/transfer_items/:id/received_quantity", server.setTransferItemReceivedQuantity)

	authRoutes.POST("/purchases", server.createPurchase)
	authRoutes.GET("/purchases/:id", server.getPurchase)
	authRoutes.GET("/purchases", server.listPurchases)
	authRoutes.POST("/purchase_items", server.addPurchaseItem)
	authRoutes.POST("/purchases/:id/receive", server.receivePurchase)
	authRoutes.POST("/purchases/:id/cancel", server.cancelPurchase)

	authRoutes.POST("/batches", server.createBatch)
	authRoutes.GET("/batches/:id", server.getBatch)
	authRoutes.GET("/batches", server.listBatches)
	authRoutes.DELETE("/batches/:id", server.deleteBatch)

	authRoutes.POST("/branches", server.createBranch)
	authRoutes.GET("/branches", server.listBranches)
	authRoutes.GET("/default_branch_settings", server.getDefaultBranchSettings)
	authRoutes.PUT("/default_branch_settings", server.putDefaultBranchSettings)
	authRoutes.POST("/default_branch_settings/apply_to_all", server.applyDefaultBranchSettingsToAll)
	authRoutes.GET("/branches/connected", server.listConnectedBranches)
	authRoutes.GET("/branches/syncing", server.listSyncingBranches)
	authRoutes.POST("/branches/:id/activate", server.setBranchActive(true))
	authRoutes.POST("/branches/:id/deactivate", server.setBranchActive(false))
	authRoutes.POST("/branches/:id/hide_from_reports", server.setBranchHiddenFromReports(true))
	authRoutes.POST("/branches/:id/show_in_reports", server.setBranchHiddenFromReports(false))
	authRoutes.GET("/branches/:id/registers", server.listBranchRegisters)
	authRoutes.POST("/branches/:id/registers", server.createBranchRegister)
	authRoutes.GET("/branch_registers", server.listAllBranchRegisters)
	authRoutes.GET("/branch_registers/connected", server.listConnectedRegisters)
	authRoutes.PUT("/branch_registers/:id", server.updateBranchRegister)
	authRoutes.POST("/branch_registers/:id/activate", server.setBranchRegisterActive(true))
	authRoutes.POST("/branch_registers/:id/deactivate", server.setBranchRegisterActive(false))
	authRoutes.POST("/branch_registers/:id/hide_from_reports", server.setBranchRegisterHiddenFromReports(true))
	authRoutes.POST("/branch_registers/:id/show_in_reports", server.setBranchRegisterHiddenFromReports(false))
	authRoutes.POST("/branch_registers/:id/rotate_key", server.rotateBranchRegisterKey)
	authRoutes.PUT("/branch_registers/:id/quick_access_mode", server.putRegisterQuickAccessMode)
	authRoutes.GET("/branches/:id/quick_access", server.getBranchQuickAccess)
	authRoutes.PUT("/branches/:id/quick_access", server.putBranchQuickAccess)

	authRoutes.GET("/branch_expenses", server.listBranchExpenses)
	authRoutes.POST("/branch_expenses", server.adminCreateBranchExpense)
	authRoutes.GET("/branch_expense_images", server.listBranchExpenseImages)
	authRoutes.GET("/branch_expenses/:id/images", server.listBranchExpenseImagesForExpense)
	authRoutes.GET("/branch_expense_images/:id/file", server.getBranchExpenseImageFile)
	authRoutes.PUT("/branch_expense_images/:id/rotation", server.rotateBranchExpenseImage)
	authRoutes.PUT("/branch_expense_images/:id/replace", server.replaceBranchExpenseImage)
	authRoutes.GET("/branch_shift_receipt_images", server.listBranchShiftReceiptImages)
	authRoutes.GET("/branch_shifts/:id/receipt_image", server.getBranchShiftReceiptImageForShift)
	authRoutes.GET("/branch_shift_receipt_images/:id/file", server.getBranchShiftReceiptImageFile)
	authRoutes.PUT("/branch_shift_receipt_images/:id/rotation", server.rotateBranchShiftReceiptImage)
	authRoutes.PUT("/branch_shift_receipt_images/:id/replace", server.replaceBranchShiftReceiptImage)
	authRoutes.GET("/branch_invoices", server.listBranchInvoices)
	authRoutes.GET("/branch_sync_conflicts", server.listBranchSyncConflicts)
	authRoutes.GET("/branch_sync_conflicts/summary", server.branchSyncConflictSummary)
	authRoutes.POST("/branch_sync_conflicts/bulk_resolve", server.bulkResolveBranchSyncConflicts)
	authRoutes.POST("/branch_sync_conflicts/:id/resolve", server.resolveBranchSyncConflict)
	authRoutes.POST("/branch_sync_conflicts/:id/dismiss", server.dismissBranchSyncConflict)
	authRoutes.GET("/branch_invoices/:id/items", server.listBranchInvoiceItems)
	authRoutes.GET("/branch_invoices/daily_income", server.dailyIncome)
	authRoutes.GET("/branch_invoices/day_summary", server.branchDaySummary)
	authRoutes.GET("/branch_sales_stats", server.listBranchSalesStats)
	authRoutes.GET("/branch_shifts", server.listBranchShifts)
	authRoutes.GET("/branch_visitor_events", server.listBranchVisitorDays)
	authRoutes.GET("/branch_visitor_stats", server.listBranchVisitorStats)
	authRoutes.GET("/busy_hours", server.getBusyHours)
	authRoutes.GET("/branch_invoice_settlements", server.listBranchInvoiceSettlements)
	authRoutes.GET("/branch_attendance_changes", server.listBranchAttendanceChanges)
	authRoutes.GET("/branch_attendance_changes/pending", server.listPendingBranchAttendanceRequests)
	authRoutes.GET("/branch_attendance_events", server.listBranchAttendanceEvents)
	authRoutes.GET("/branch_attendance_events/range", server.listBranchAttendanceEventsInRange)
	authRoutes.POST("/branch_attendance_events/import", server.importBranchAttendanceEventsCSV)
	authRoutes.PUT("/branch_attendance_events/:id", server.updateBranchAttendanceEvent)
	authRoutes.DELETE("/branch_attendance_events/:id", server.deleteBranchAttendanceEvent)

	authRoutes.GET("/dashboard/summary", server.getDashboardSummary)
	authRoutes.GET("/dashboard/sales", server.listDashboardSales)
	authRoutes.GET("/dashboard/purchases", server.listDashboardPurchases)
	authRoutes.GET("/dashboard/expenses", server.listDashboardExpenses)
	authRoutes.GET("/dashboard/activities", server.listDashboardActivities)
	authRoutes.GET("/sales_dashboard", server.getSalesDashboard)
	authRoutes.GET("/sales_analysis", server.getSalesAnalysis)
	authRoutes.GET("/financials", server.getFinancials)
	authRoutes.GET("/branches/:id/settings", server.getBranchSettings)
	authRoutes.PUT("/branches/:id/settings/managed_locally", server.putBranchSettingsManagedLocally)
	authRoutes.POST("/branches/:id/commands", server.createBranchCommand)
	authRoutes.GET("/branches/:id/commands", server.listBranchCommands)
	authRoutes.GET("/branch_commands", server.listAllBranchCommands)
	authRoutes.GET("/branch_commands/summary", server.branchCommandSummary)
	authRoutes.POST("/branch_commands/:id/cancel", server.cancelBranchCommand)
	authRoutes.POST("/branch_commands/:id/retry", server.retryBranchCommand)

	authRoutes.GET("/branches/:id/payment_methods", server.listBranchPaymentMethods)
	authRoutes.PUT("/branches/:id/payment_methods/:account_id", server.putBranchPaymentMethod)

	authRoutes.GET("/branches/:id/targets", server.listBranchTargets)
	authRoutes.POST("/branches/:id/targets", server.createBranchTarget)
	authRoutes.PUT("/branch_targets/:id", server.updateBranchTarget)
	authRoutes.DELETE("/branch_targets/:id", server.deleteBranchTarget)

	authRoutes.GET("/branches/:id/target_series", server.listBranchTargetSeries)
	authRoutes.POST("/branches/:id/target_series", server.createBranchTargetSeries)
	authRoutes.PUT("/target_series/:id", server.updateBranchTargetSeries)
	authRoutes.PUT("/target_series/:id/active", server.setBranchTargetSeriesActive)
	authRoutes.DELETE("/target_series/:id", server.deleteBranchTargetSeries)

	authRoutes.GET("/branches/:id/salespersons", server.listSalespersons)
	authRoutes.POST("/branches/:id/salespersons", server.createSalesperson)
	authRoutes.PUT("/salespersons/:id", server.updateSalesperson)
	authRoutes.PUT("/salespersons/:id/active", server.setSalespersonActive)
	authRoutes.PUT("/salespersons/:id/color", server.setSalespersonColor)
	authRoutes.PUT("/salespersons/:id/attendance_id", server.setSalespersonAttendanceID)
	authRoutes.PUT("/salespersons/:id/is_salesperson", server.setSalespersonIsSalesperson)

	authRoutes.GET("/employees", server.listEmployees)
	authRoutes.POST("/employees", server.createEmployee)
	authRoutes.PUT("/employees/salaries", server.batchUpdateEmployeeSalaries)
	authRoutes.GET("/employees/:id", server.getEmployee)
	authRoutes.PUT("/employees/:id", server.updateEmployee)
	authRoutes.DELETE("/employees/:id", server.deleteEmployee)

	authRoutes.GET("/branches/:id/branch_users", server.listBranchUsers)
	authRoutes.POST("/branches/:id/branch_users", server.createBranchUser)
	authRoutes.PUT("/branch_users/:id", server.updateBranchUser)
	authRoutes.PUT("/branch_users/:id/pin", server.setBranchUserPin)
	authRoutes.DELETE("/branch_users/:id", server.deleteBranchUser)

	// Till app updates -- App Updates section, see api/app_update.go.
	authRoutes.GET("/app_releases", server.listAppReleases)
	authRoutes.POST("/app_releases", server.createAppRelease)
	authRoutes.PUT("/app_releases/:id", server.updateAppRelease)
	authRoutes.DELETE("/app_releases/:id", server.deleteAppRelease)
	authRoutes.GET("/app_updates/registers", server.listAppUpdateRegisters)
	authRoutes.PUT("/app_updates/target", server.setAppUpdateTarget)
	authRoutes.GET("/app_updates/events", server.listAppUpdateEvents)

	// adminWS can't sit under authRoutes: authMiddleware only reads the
	// Authorization header, and a browser WebSocket handshake can't set
	// custom headers. Auth happens inside adminWS itself via a query-param
	// token instead.
	router.GET("/admin/ws", server.adminWS)

	// Security page (admin-only: not in access.go's routeRules).
	authRoutes.GET("/security/overview", server.getSecurityOverview)
	authRoutes.GET("/security/alerts/open_count", server.countOpenSecurityAlerts)
	authRoutes.GET("/security/visitors", server.listSecurityVisitors)
	authRoutes.POST("/security/alerts/acknowledge", server.acknowledgeSecurityAlerts)
	authRoutes.PUT("/security/settings", server.updateSecuritySettings)
	authRoutes.POST("/security/bans", server.banIP)
	authRoutes.DELETE("/security/bans/:ip", server.unbanIP)

	// branchRoutes is authenticated with a per-register API key rather than
	// a PASETO user token — see branchAuthMiddleware. Kept under its own path
	// prefix so branch-facing request/response shapes (branch_id, client_ref
	// idempotency, etc.) never collide with the human-operator endpoints above.
	branchRoutes := router.Group("/branch").Use(server.branchAuthMiddleware())
	branchRoutes.GET("/health", server.branchHealth)
	branchRoutes.POST("/sales_invoices", server.createBranchSalesInvoice)
	branchRoutes.POST("/invoice_salespersons", server.setBranchInvoiceSalespersons)
	branchRoutes.POST("/return_invoices", server.createBranchReturnInvoice)
	branchRoutes.POST("/exchange_invoices", server.createBranchExchangeInvoice)
	branchRoutes.POST("/expenses", server.createBranchExpense)
	branchRoutes.POST("/loans", server.createBranchLoan)
	branchRoutes.POST("/shift_closes", server.createBranchShiftClose)
	branchRoutes.POST("/whatsapp/thank_you", server.sendBranchWhatsAppThankYou)
	branchRoutes.POST("/visitor_events", server.createBranchVisitorEvent)
	branchRoutes.POST("/visitor_events_batch", server.createBranchVisitorEvents)
	branchRoutes.POST("/invoice_settlements", server.createBranchInvoiceSettlement)
	branchRoutes.POST("/attendance_events", server.createBranchAttendanceEvents)
	branchRoutes.POST("/attendance_changes", server.createBranchAttendanceChange)
	branchRoutes.GET("/sync/changes", server.branchSyncChanges)
	branchRoutes.POST("/sync/reconcile", server.branchReconcile)
	branchRoutes.POST("/sync/full_sync/start", server.branchFullSyncStart)
	branchRoutes.POST("/sync/full_sync/end", server.branchFullSyncEnd)
	branchRoutes.POST("/quick_access/mode", server.branchSetQuickAccessMode)
	branchRoutes.POST("/quick_access/import", server.branchImportQuickAccess)
	branchRoutes.POST("/sync/ids", server.branchSyncIDs)
	branchRoutes.POST("/sync/fetch", server.branchSyncFetch)
	branchRoutes.GET("/sync/client_uids", server.branchClientUIDs)
	branchRoutes.GET("/ws", server.branchWS)
	branchRoutes.PUT("/settings", server.putBranchSettings)
	branchRoutes.POST("/clients", server.putBranchClient)
	branchRoutes.POST("/clients_batch", server.putBranchClients)
	branchRoutes.GET("/targets/progress", server.branchTargetsProgress)
	branchRoutes.GET("/commands/pending", server.branchPendingCommands)
	branchRoutes.POST("/commands/:id/ack", server.ackBranchCommand)
	branchRoutes.GET("/expenses/upload_status", server.branchExpenseUploadStatus)
	branchRoutes.GET("/expense_photos", server.branchExpensePhotos)
	branchRoutes.GET("/expense_photos/:id/file", server.branchExpensePhotoFile)
	branchRoutes.GET("/transfer_photo", server.branchTransferPhoto)
	branchRoutes.GET("/loan_photos", server.branchLoanPhotos)
	branchRoutes.GET("/loan_photos/:id/file", server.branchLoanPhotoFile)
	branchRoutes.POST("/expenses/upload_link", server.branchExpenseUploadLink)
	branchRoutes.POST("/loans/upload_link", server.branchLoanUploadLink)
	branchRoutes.GET("/shift_closes/upload_status", server.branchShiftUploadStatus)
	branchRoutes.POST("/shift_receipt_images", server.createBranchShiftReceiptImage)
	branchRoutes.POST("/transfer_requests", server.createBranchTransferRequest)
	branchRoutes.POST("/transfer_requests/:id/confirm", server.confirmBranchTransfer)
	branchRoutes.GET("/transfer_requests/upload_status", server.branchTransferUploadStatus)
	branchRoutes.GET("/app_update", server.branchAppUpdate)
	branchRoutes.GET("/app_update/download/:id", server.branchAppUpdateDownload)
	branchRoutes.POST("/app_update/status", server.branchAppUpdateStatus)

	// expense_uploads is deliberately public: a phone scanning the QR code
	// kashi-pos shows has no kashi login and shouldn't need one. Every
	// handler here re-validates its own one-time token instead of relying
	// on any auth middleware -- see expense_upload.go.
	router.GET("/expense_uploads/:token", server.expenseUploadPage)
	router.GET("/expense_uploads/:token/status", server.expenseUploadStatus)
	router.POST("/expense_uploads/:token/images", server.uploadExpenseImages)
	// The same page and flow for a branch loan's photos (loan_upload.go).
	router.GET("/loan_uploads/:token", server.expenseUploadPage)
	router.GET("/loan_uploads/:token/status", server.loanUploadStatus)
	router.POST("/loan_uploads/:token/images", server.uploadLoanImages)

	// shift_uploads is the same public, token-gated flow for the one
	// close-shift receipt photo -- see shift_upload.go.
	router.GET("/shift_uploads/:token", server.shiftUploadPage)
	router.GET("/shift_uploads/:token/status", server.shiftUploadStatus)
	router.POST("/shift_uploads/:token/images", server.uploadShiftReceiptImage)

	// transfer_uploads is the same public, token-gated flow for the one
	// optional transfer photo -- see transfer_upload.go.
	router.GET("/transfer_uploads/:token", server.transferUploadPage)
	router.GET("/transfer_uploads/:token/status", server.transferUploadStatus)
	router.POST("/transfer_uploads/:token/images", server.uploadTransferAttachment)

	server.router = router
}

func (server *Server) Start(address string) error {
	go server.expireSyncLeasesLoop()
	go server.listenForSyncChanges()
	return server.router.Run(address)
}
