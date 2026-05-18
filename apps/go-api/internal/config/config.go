package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL        string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string
	Port               string
	R2AccountID        string
	R2AccessKeyID      string
	R2SecretAccessKey  string
	R2Bucket           string
	R2Endpoint         string
	AgentPhoneAPIKey   string
	AgentPhoneAgentID  string

	AgentPhoneWebhookSecret string
	AgentPhoneWebhookStream string
	AgentPhoneLLMModel      string
	SubAgentLLMModel        string
	AgentPhoneMaxTurns      int
	AgentPhoneToolLoopMax   int
	AIGatewayAPIKey         string
	AIGatewayBaseURL        string
	RedisURL                string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	c := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		FrontendURL:        os.Getenv("FRONTEND_URL"),
		Port:               os.Getenv("PORT"),
		R2AccountID:        os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:      os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey:  os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:           os.Getenv("R2_BUCKET"),
		R2Endpoint:         os.Getenv("R2_ENDPOINT"),
		AgentPhoneAPIKey:   os.Getenv("AGENTPHONE_API_KEY"),
		AgentPhoneAgentID:  os.Getenv("AGENTPHONE_AGENT_ID"),

		AgentPhoneWebhookSecret: os.Getenv("AGENTPHONE_WEBHOOK_SECRET"),
		AgentPhoneWebhookStream: os.Getenv("AGENTPHONE_WEBHOOK_STREAM"),
		AgentPhoneLLMModel:      os.Getenv("AGENTPHONE_LLM_MODEL"),
		SubAgentLLMModel:        os.Getenv("SUBAGENT_LLM_MODEL"),
		AgentPhoneMaxTurns:      atoiOr(os.Getenv("AGENTPHONE_MAX_TURNS"), 40),
		AgentPhoneToolLoopMax:   atoiOr(os.Getenv("AGENTPHONE_TOOL_LOOP_MAX"), 5),
		AIGatewayAPIKey:         os.Getenv("AI_GATEWAY_API_KEY"),
		AIGatewayBaseURL:        os.Getenv("AI_GATEWAY_BASE_URL"),
		RedisURL:                os.Getenv("REDIS_URL"),
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.AgentPhoneWebhookStream == "" {
		c.AgentPhoneWebhookStream = "off"
	}
	if c.AgentPhoneLLMModel == "" {
		c.AgentPhoneLLMModel = "openai/gpt-4o-mini"
	}
	if c.SubAgentLLMModel == "" {
		c.SubAgentLLMModel = "anthropic/claude-haiku-4-5"
	}
	if c.AIGatewayBaseURL == "" {
		c.AIGatewayBaseURL = "https://ai-gateway.vercel.sh/v1"
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}
	return c, nil
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
