package util

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	GRPCServerAddress    string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	GatewayServerAddress string        `mapstructure:"GATEWAY_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"access_token_duration"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	// ExpenseImageStorageDir is where receipt photos uploaded through the
	// one-time QR flow (api/expense_upload.go) are saved on local disk.
	ExpenseImageStorageDir string `mapstructure:"EXPENSE_IMAGE_STORAGE_DIR"`
	// ShiftReceiptImageStorageDir is where close-shift receipt photos
	// forwarded by kashi-pos (api/branch_shift_receipt_image.go) are saved.
	// Defaults to ./storage/shift_receipt_images when unset.
	ShiftReceiptImageStorageDir string `mapstructure:"SHIFT_RECEIPT_IMAGE_STORAGE_DIR"`
	// TransferAttachmentStorageDir is where the optional photo attached to
	// a branch-initiated transfer request (api/branch_transfer.go) is
	// saved. Defaults to ./storage/transfer_attachments when unset.
	TransferAttachmentStorageDir string `mapstructure:"TRANSFER_ATTACHMENT_STORAGE_DIR"`
	// AppReleaseStorageDir is where till app builds published for remote
	// update (api/app_update.go) are saved. Defaults to
	// ./storage/app_releases when unset.
	AppReleaseStorageDir string `mapstructure:"APP_RELEASE_STORAGE_DIR"`
	// GeoIPDir holds the DB-IP lite databases the Security page locates
	// visitors with (api/geoip.go downloads them). Defaults to
	// ./storage/geoip when unset.
	GeoIPDir string `mapstructure:"GEOIP_DIR"`
	// AppLogPath is the bounded, rotating backend log file. It defaults to
	// logs/kashi.log when APP_LOG_PATH is not set.
	AppLogPath string `mapstructure:"APP_LOG_PATH"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&config)
	return
}
