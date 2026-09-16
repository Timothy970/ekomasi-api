package config

import (
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

type ServerConfig struct {
	Port        string
	Environment string
	BaseURL     string
	FrontEndURL string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       int
	TTL      int
}

type JWTConfig struct {
	Secret string
}

type MpesaConfig struct {
	ConsumerKey         string
	ConsumerSecret      string
	ShortCode           string
	Passkey             string
	CallbackURL         string
	SendURL             string
	InitiatorName       string
	InitiatorPassword   string
	SecurityCredentials string
	ReturnURL           string
	StatusURL           string
}

type WhatsAppConfig struct {
	SendURL        string
	Sender         string
	OpenWAURL      string
	FlowPrivateKey string
}

type SMSConfig struct {
	APIKey    string
	PartnerID string
	Shortcode string
	V2URL     string
}

type PaymentGatewayConfig struct {
	PaystackSecretKey    string
	PaystackPublicKey    string
	FlutterwaveSecretKey string
	FlutterwavePublicKey string
	StripeSecretKey      string
	StripePublicKey      string
}
type OpenWAConfig struct {
	APIKey  string
	BaseURL string
	Session string
}

type WhatsAppCloudConfig struct {
	PhoneNumberID      string
	WABAID             string
	AccessToken        string
	WebhookVerifyToken string
	AppSecret          string
	FlowPrivateKey     string
	FlowPassphrase     string
}

type StorageConfig struct {
	Bucket              string
	FlociEndpoint       string
	UseCloudinary       bool
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
}

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	JWT            JWTConfig
	Mpesa          MpesaConfig
	WhatsApp       WhatsAppConfig
	WhatsAppCloud  WhatsAppCloudConfig
	SMS            SMSConfig
	PaymentGateway PaymentGatewayConfig
	OpenWA         OpenWAConfig
	Storage        StorageConfig
}

var (
	AppConfig *Config
	once      sync.Once
)

// LoadConfig initializes and returns the global AppConfig singleton
func LoadConfig(envFiles ...string) *Config {
	once.Do(func() {
		if len(envFiles) > 0 {
			if err := godotenv.Load(envFiles...); err != nil {
				log.Println("Warning: Could not load specified .env file(s):", err)
			}
		} else {
			if err := godotenv.Load(); err != nil {
				log.Println("Warning: Could not load default .env file:", err)
			}
		}

		AppConfig = &Config{
			Server: ServerConfig{
				Port:        getEnv("PORT", "8000"),
				Environment: getEnv("ENVIRONMENT", "development"),
				BaseURL:     getEnv("BASE_URL", "http://localhost:8000"),
				FrontEndURL: getEnv("FRONT_END_BASE_URL", "http://localhost:3000"),
			},
			Database: DatabaseConfig{
				Host:     getEnv("MYSQL_HOST", getEnv("DB_HOST", "127.0.0.1")),
				Port:     getEnv("MYSQL_PORT", getEnv("DB_PORT", "3306")),
				User:     getEnv("MYSQL_USER", getEnv("DB_USER", "root")),
				Password: getEnv("MYSQL_PASS", getEnv("DB_PASSWORD", "")),
				Name:     getEnv("DB_NAME", "ekomasi"),
			},
			Redis: RedisConfig{
				Host:     getEnv("REDIS_HOST", "127.0.0.1"),
				Port:     getEnv("REDIS_PORT", "6379"),
				User:     getEnv("REDIS_USER", ""),
				Password: getEnv("REDIS_PASS", ""),
				DB:       getEnvAsInt("REDIS_DB", 0),
				TTL:      getEnvAsInt("REDIS_TIME", 30),
			},
			JWT: JWTConfig{
				Secret: getEnv("JWT_SECRET", "secret"),
			},
			Mpesa: MpesaConfig{
				ConsumerKey:         getEnv("MPESA_CONSUMER_KEY", ""),
				ConsumerSecret:      getEnv("MPESA_CONSUMER_SECRET", ""),
				ShortCode:           getEnv("MPESA_SHORTCODE", ""),
				Passkey:             getEnv("MPESA_PASSKEY", ""),
				CallbackURL:         getEnv("MPESA_CALLBACK_URL", ""),
				SendURL:             getEnv("MPESA_SEND_URL", "https://sandbox.safaricom.co.ke/"),
				InitiatorName:       getEnv("MPESA_INITIATOR_NAME", ""),
				InitiatorPassword:   getEnv("MPESA_INITIATOR_PASSWORD", ""),
				SecurityCredentials: getEnv("MPESA_SECURITY_CREDENTIALS", ""),
				ReturnURL:           getEnv("MPESA_RETURN_URL", ""),
				StatusURL:           getEnv("MPESA_STATUS_URL", ""),
			},
			WhatsApp: WhatsAppConfig{
				SendURL:        getEnv("WHATSAPPSENDURL", ""),
				Sender:         getEnv("WHATSAPPSENDER", ""),
				OpenWAURL:      getEnv("OPENWA_URL", "http://localhost:2785"),
				FlowPrivateKey: getEnv("WHATSAPP_FLOW_PRIVATE_KEY", ""),
			},
			SMS: SMSConfig{
				APIKey:    getEnv("SMSAPIKEY", getEnv("APIKEY", "")),
				PartnerID: getEnv("SMSPARTNERID", getEnv("PARTNERID", "")),
				Shortcode: getEnv("SHORTCODE", "Emalify"),
				V2URL:     getEnv("V2_URL", "https://api.v2.emalify.com/api"),
			},
			PaymentGateway: PaymentGatewayConfig{
				PaystackSecretKey:    getEnv("PAYSTACK_SECRET_KEY", ""),
				PaystackPublicKey:    getEnv("PAYSTACK_PUBLIC_KEY", ""),
				FlutterwaveSecretKey: getEnv("FLUTTERWAVE_SECRET_KEY", ""),
				FlutterwavePublicKey: getEnv("FLUTTERWAVE_PUBLIC_KEY", ""),
				StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
				StripePublicKey:      getEnv("STRIPE_PUBLIC_KEY", ""),
			},
			OpenWA: OpenWAConfig{
				BaseURL: getEnv("OPENWA_URL", "http://localhost:8090"),
				APIKey:  getEnv("OPENWA_API_KEY", "api-key"),
				Session: getEnv("OPENWA_SESSION", "my-session"),
			},
			WhatsAppCloud: WhatsAppCloudConfig{
				PhoneNumberID:      getEnv("WHATSAPP_PHONE_NUMBER_ID", ""),
				WABAID:             getEnv("WHATSAPP_WABA_ID", ""),
				AccessToken:        getEnv("WHATSAPP_ACCESS_TOKEN", ""),
				WebhookVerifyToken: getEnv("WHATSAPP_WEBHOOK_VERIFY_TOKEN", ""),
				AppSecret:          getEnv("WHATSAPP_APP_SECRET", ""),
				FlowPrivateKey:     getEnv("WHATSAPP_FLOW_PRIVATE_KEY", ""),
				FlowPassphrase:     getEnv("WHATSAPP_FLOW_PASSPHRASE", ""),
			},
			Storage: StorageConfig{
				Bucket:              getEnv("STORAGE_BUCKET", getEnv("BUCKET_NAME", "development-ecommerce-api-images")),
				FlociEndpoint:       getEnv("FLOCI_ENDPOINT", ""),
				UseCloudinary:       getEnvAsBool("USE_CLOUDINARY", false),
				CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
				CloudinaryAPIKey:    getEnv("CLOUDINARY_API_KEY", ""),
				CloudinaryAPISecret: getEnv("CLOUDINARY_API_SECRET", ""),
			},
		}

		log.Println("Configuration loaded successfully")
	})

	return AppConfig
}

// Get returns the loaded configuration singleton
func Get() *Config {
	if AppConfig == nil {
		return LoadConfig()
	}
	return AppConfig
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultVal
}
