package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    // Existing fields
    Port string
    
    // HTTP Client Configuration
    MaxConnections       int
    KeepAliveConnections int
    ConnectionTimeout    time.Duration
    FirstTokenTimeout    time.Duration
    MultimodalFirstTokenTimeout time.Duration // Extended timeout for multimodal requests
    
    // API Configuration
    HiddenModels []string
    ProxyAPIKey  string
    AdminAPIKey  string // Admin API key for management endpoints
    
    // API Key Storage
    APIKeyStorageDir string // Directory for storing API keys
    
    // VPN/Proxy Configuration
    VPNProxyURL string // HTTP/HTTPS/SOCKS5 proxy URL (e.g., http://127.0.0.1:7890, socks5://127.0.0.1:1080)
    
    // AWS Configuration
    AWSRegion   string
    AWSProfile  string
    EnableSigV4 bool
    
    // SSO Configuration for SigV4 mode
    // When EnableSigV4 is true and UseSSOCredentials is true,
    // bearer tokens are converted to IAM credentials via SSO API
    UseSSOCredentials bool   // Enable SSO-derived credentials for SigV4
    SSOAccountID      string // AWS account ID for SSO
    SSORoleName       string // SSO role name
    
    // Headless Mode Configuration
    HeadlessMode bool   // Enable headless OIDC authentication (no AWS CLI required)
    SSOStartURL  string // SSO start URL for headless mode
    SSORegion    string // SSO region for headless mode
    
    // Optional: Pre-registered OIDC client (for production deployments)
    SSOClientID     string // Pre-registered OIDC client ID
    SSOClientSecret string // Pre-registered OIDC client secret
    SSOClientExpiry int64  // Client secret expiry timestamp
    
    // Optional: Browser Automation (for fully automated auth)
    AutomateAuth bool   // Enable browser automation
    SSOUsername  string // IAM Identity Center username
    SSOPassword  string // IAM Identity Center password
    MFATOTPSecret string // TOTP secret for automated MFA (base32 encoded)
    
    // OIDC Configuration (legacy, kept for backward compatibility)
    OIDCStartURL string
    OIDCRegion   string
    
    // Profile Configuration
    ProfileARN string
    
    // Opt-out Configuration
    OptOutTelemetry bool
    
    // Retry Configuration
    MaxRetries int
    MaxBackoff time.Duration
    
    // Timeout Configuration
    ConnectTimeout   time.Duration
    ReadTimeout      time.Duration
    OperationTimeout time.Duration
    
    // Stream Protection
    StalledStreamGrace time.Duration
    
    // Multimodal API Configuration
    UseQDeveloper bool   // Use Q endpoint with SigV4 instead of CodeWhisperer with bearer
    APIEndpoint   string // Override default endpoint (optional)
    
    // Beta Features
    BetaFeatures BetaFeatures
    
    // Conversation Context Cleanup Configuration
    ContextCleanupInterval time.Duration // How often to run cleanup (default: 5 minutes)
    ContextMaxAge          time.Duration // Maximum age before context is considered stale (default: 30 minutes)
    
    // Response Cache Configuration
    ResponseCacheEnabled bool          // Enable response caching (default: true)
    ResponseCacheTTL     time.Duration // Cache TTL (default: 5 minutes)
    ResponseCacheSize    int           // Max cache entries (default: 1000)
    
    // Request Deduplication Configuration
    RequestDedupEnabled bool // Enable request deduplication (default: true)
}

func Load() *Config {
    // Determine if we should use QDeveloper mode
    // Q_USE_SENDMESSAGE env var triggers QDeveloper mode (like CloudShell)
    useQDeveloper := getBoolEnv("Q_USE_SENDMESSAGE", false) || 
                     getBoolEnv("USE_Q_DEVELOPER", false)
    
    return &Config{
        Port:                 getEnv("PORT", "8080"),
        MaxConnections:       getIntEnv("MAX_CONNECTIONS", 100),
        KeepAliveConnections: getIntEnv("KEEP_ALIVE_CONNECTIONS", 10),
        ConnectionTimeout:    getDurationEnv("CONNECTION_TIMEOUT", 10*time.Second),
        FirstTokenTimeout:    getDurationEnv("FIRST_TOKEN_TIMEOUT", 120*time.Second),
        MultimodalFirstTokenTimeout: getDurationEnv("MULTIMODAL_FIRST_TOKEN_TIMEOUT", 180*time.Second),
        HiddenModels:         []string{},
        ProxyAPIKey:          getEnv("PROXY_API_KEY", ""),
        AdminAPIKey:          getEnvOrSecret("ADMIN_API_KEY", "admin_api_key", ""),
        APIKeyStorageDir:     getEnv("API_KEY_STORAGE_DIR", ".kiro/api-keys"),
        VPNProxyURL:          getEnv("VPN_PROXY_URL", ""),
        AWSRegion:            getEnv("AWS_REGION", "us-east-1"),
        AWSProfile:           getEnv("AWS_PROFILE", ""),
        EnableSigV4:          getBoolEnv("AMAZON_Q_SIGV4", false),
        UseSSOCredentials:    getBoolEnv("USE_SSO_CREDENTIALS", false),
        SSOAccountID:         getEnv("AWS_SSO_ACCOUNT_ID", ""),
        SSORoleName:          getEnv("AWS_SSO_ROLE_NAME", ""),
        HeadlessMode:         getBoolEnv("HEADLESS_MODE", false),
        SSOStartURL:          getEnv("SSO_START_URL", ""),
        SSORegion:            getEnv("SSO_REGION", getEnv("AWS_REGION", "us-east-1")),
        SSOClientID:          getEnv("SSO_CLIENT_ID", ""),
        SSOClientSecret:      getEnv("SSO_CLIENT_SECRET", ""),
        SSOClientExpiry:      getInt64Env("SSO_CLIENT_EXPIRY", 0),
        AutomateAuth:         getBoolEnv("AUTOMATE_AUTH", false),
        SSOUsername:          getEnv("SSO_USERNAME", ""),
        SSOPassword:          getEnvOrSecret("SSO_PASSWORD", "sso_password", ""),
        MFATOTPSecret:        getEnvOrSecret("MFA_TOTP_SECRET", "mfa_totp_secret", ""),
        OIDCStartURL:         getEnv("OIDC_START_URL", ""),
        OIDCRegion:           getEnv("OIDC_REGION", "us-east-1"),
        ProfileARN:           getEnv("PROFILE_ARN", ""),
        OptOutTelemetry:      getBoolEnv("OPT_OUT_TELEMETRY", false),
        MaxRetries:           getIntEnv("MAX_RETRIES", 3),
        MaxBackoff:           getDurationEnv("MAX_BACKOFF", 30*time.Second),
        ConnectTimeout:       getDurationEnv("CONNECT_TIMEOUT", 10*time.Second),
        ReadTimeout:          getDurationEnv("READ_TIMEOUT", 30*time.Second),
        OperationTimeout:     getDurationEnv("OPERATION_TIMEOUT", 60*time.Second),
        StalledStreamGrace:   getDurationEnv("STALLED_STREAM_GRACE", 5*time.Minute),
        UseQDeveloper:        useQDeveloper,
        APIEndpoint:          getEnv("AMAZON_Q_ENDPOINT", ""),
        BetaFeatures:         LoadBetaFeatures(),
        ContextCleanupInterval: getDurationEnv("CONTEXT_CLEANUP_INTERVAL", 5*time.Minute),
        ContextMaxAge:          getDurationEnv("CONTEXT_MAX_AGE", 30*time.Minute),
        ResponseCacheEnabled:   getBoolEnv("RESPONSE_CACHE_ENABLED", true),
        ResponseCacheTTL:       getDurationEnv("RESPONSE_CACHE_TTL", 5*time.Minute),
        ResponseCacheSize:      getIntEnv("RESPONSE_CACHE_SIZE", 1000),
        RequestDedupEnabled:    getBoolEnv("REQUEST_DEDUP_ENABLED", true),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// getEnvOrSecret tries to get value from environment variable first,
// then falls back to Docker secret if available
func getEnvOrSecret(key, secretName, defaultValue string) string {
    // Try environment variable first
    if value := os.Getenv(key); value != "" {
        return value
    }
    
    // Try Docker secret
    if secretName != "" {
        if data, err := os.ReadFile("/run/secrets/" + secretName); err == nil {
            return string(data)
        }
    }
    
    return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    if value == "true" {
        return true
    }
    if value == "false" {
        return false
    }
    // Invalid value, return default
    return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
    if value := os.Getenv(key); value != "" {
        // Simple comma-separated parsing
        if value == "" {
            return defaultValue
        }
        return []string{value} // For simplicity, just return single value
    }
    return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
            return intValue
        }
    }
    return defaultValue
}
