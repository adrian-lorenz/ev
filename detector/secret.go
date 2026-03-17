package detector

import "regexp"

type secretRule struct {
	id          string
	pattern     *regexp.Regexp
	secretGroup int
	confidence  float64
}

var secretRules []secretRule

func init() {
	type raw struct {
		id    string
		pat   string
		group int
		sev   string
	}
	sev := map[string]float64{
		"CRITICAL": 1.0, "HIGH": 0.9, "MEDIUM": 0.75, "LOW": 0.6, "WARNING": 0.5,
	}
	rules := []raw{
		// Cloud / VCS
		{"aws-access-key", `(?i)(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`, 0, "CRITICAL"},
		{"aws-secret-key", `(?i)aws[_\-\s\.]{0,5}secret[_\-\s\.]{0,5}(access[_\-\s\.]{0,5})?key["'\s]*[:=]["'\s]*([A-Za-z0-9+/]{40})`, 2, "CRITICAL"},
		{"github-pat", `(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,255}`, 0, "CRITICAL"},
		{"github-fine-grained-pat", `github_pat_[A-Za-z0-9_]{82}`, 0, "CRITICAL"},
		{"gitlab-pat", `glpat-[A-Za-z0-9\-]{20}`, 0, "CRITICAL"},
		{"google-api-key", `AIza[0-9A-Za-z\-_]{35}`, 0, "HIGH"},
		{"google-oauth-client", `GOCSPX-[A-Za-z0-9\-_]{28}`, 0, "HIGH"},
		{"stripe-secret", `sk_(live|test)_[A-Za-z0-9]{24,}`, 0, "CRITICAL"},
		{"stripe-publishable", `pk_(live|test)_[A-Za-z0-9]{24,}`, 0, "LOW"},
		{"slack-token", `xox[baprs]-([0-9a-zA-Z]{10,48})`, 0, "HIGH"},
		{"slack-webhook", `https://hooks\.slack\.com/services/T[A-Z0-9]+/B[A-Z0-9]+/[A-Za-z0-9]+`, 0, "HIGH"},
		{"sendgrid-api", `SG\.[A-Za-z0-9\-_]{22}\.[A-Za-z0-9\-_]{43}`, 0, "HIGH"},
		{"twilio-account-sid", `AC[a-z0-9]{32}`, 0, "HIGH"},
		{"jwt-token", `eyJ[A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+\.?[A-Za-z0-9\-_.+/=]*`, 0, "MEDIUM"},
		{"private-key-header", `-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY( BLOCK)?-----`, 0, "CRITICAL"},
		{"generic-secret", `(?i)(secret|password|passwd|pwd|api[_-]?key|auth[_-]?token|access[_-]?token)["'\s]*[:=]["'\s]+([A-Za-z0-9!@#$%^&*()\-_+=]{16,})`, 2, "MEDIUM"},
		{"basic-auth-url", `[a-zA-Z][a-zA-Z0-9+\-.]*://[^:@\s]+:[^:@\s]+@[^@\s]+`, 0, "HIGH"},
		{"npm-token", `npm_[A-Za-z0-9]{36}`, 0, "HIGH"},
		{"docker-hub-pat", `dckr_pat_[A-Za-z0-9\-_]{27}`, 0, "HIGH"},
		// LLM / AI
		{"openai-api-key", `sk-(?:proj-|svcacct-)?[A-Za-z0-9\-_]{32,}T3BlbkFJ[A-Za-z0-9\-_]{20,}`, 0, "CRITICAL"},
		{"openai-api-key-new", `sk-proj-[A-Za-z0-9\-_]{50,}`, 0, "CRITICAL"},
		{"anthropic-api-key", `sk-ant-(?:api03-)?[A-Za-z0-9\-_]{32,}`, 0, "CRITICAL"},
		{"cohere-api-key", `(?i)(?:cohere[._-]?(?:api[._-]?)?key|CO_API_KEY)\s*[=:]\s*([A-Za-z0-9]{40})`, 1, "CRITICAL"},
		{"mistral-api-key", `(?i)(?:mistral[._-]?(?:api[._-]?)?key|MISTRAL_API_KEY)\s*[=:]\s*([A-Za-z0-9]{32})`, 1, "CRITICAL"},
		{"huggingface-token", `hf_[A-Za-z0-9]{32,}`, 0, "HIGH"},
		{"huggingface-token-env", `(?i)(?:HUGGING_FACE|HUGGINGFACE)[._-]?(?:HUB[._-]?)?TOKEN\s*[=:]\s*([A-Za-z0-9_\-]{20,})`, 1, "HIGH"},
		{"replicate-api-key", `r8_[A-Za-z0-9]{32,}`, 0, "HIGH"},
		{"together-ai-key", `(?i)TOGETHER[._-]?API[._-]?KEY\s*[=:]\s*([A-Za-z0-9]{64})`, 1, "CRITICAL"},
		{"perplexity-api-key", `pplx-[A-Za-z0-9]{48}`, 0, "HIGH"},
		{"groq-api-key", `gsk_[A-Za-z0-9]{52}`, 0, "HIGH"},
		{"xai-api-key", `xai-[A-Za-z0-9]{32,}`, 0, "CRITICAL"},
		{"azure-openai-key", `(?i)(?:AZURE[._-]?OPENAI[._-]?(?:API[._-]?)?KEY)\s*[=:]\s*([a-f0-9]{32})`, 1, "CRITICAL"},
		{"stability-ai-key", `sk-[A-Za-z0-9]{48}\b`, 0, "HIGH"},
		// Azure / Entra / M365
		{"azure-tenant-id", `(?i)(?:tenant[_-]?id|AZURE_TENANT_ID|tenantId)\s*[=:]\s*([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`, 1, "MEDIUM"},
		{"azure-client-id", `(?i)(?:client[_-]?id|app[_-]?id|AZURE_CLIENT_ID|clientId|applicationId)\s*[=:]\s*([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`, 1, "MEDIUM"},
		{"azure-client-secret", `(?i)(?:client[_-]?secret|AZURE_CLIENT_SECRET|clientSecret)\s*[=:]\s*([A-Za-z0-9~._\-]{34,})`, 1, "CRITICAL"},
		{"azure-subscription-key", `(?i)(?:Ocp-Apim-Subscription-Key|subscription[_-]?key|APIM[_-]?KEY)\s*[=:]\s*([a-f0-9]{32})`, 1, "CRITICAL"},
		{"azure-storage-account-key", `(?i)(?:AccountKey|AZURE_STORAGE_KEY|storageAccountKey)\s*[=:]\s*([A-Za-z0-9+/]{86}==)`, 1, "CRITICAL"},
		{"azure-storage-connection-string", `DefaultEndpointsProtocol=https?;AccountName=[^;]+;AccountKey=[A-Za-z0-9+/]{86}==[^;]*`, 0, "CRITICAL"},
		{"azure-sas-token", `(?:sv|sig|se|sp)=[A-Za-z0-9%+/=&\-]{8,}(?:&(?:sv|sig|se|sp|spr|srt|ss)=[A-Za-z0-9%+/=&\-]{4,}){3,}`, 0, "CRITICAL"},
		{"azure-function-key", `(?i)(?:x-functions-key|AZURE_FUNCTION_KEY|functionKey)\s*[=:]\s*([A-Za-z0-9/+]{40,}={0,2})`, 1, "HIGH"},
		{"azure-service-bus-connstr", `Endpoint=sb://[^;]+\.servicebus\.windows\.net/;SharedAccessKeyName=[^;]+;SharedAccessKey=[A-Za-z0-9+/]{43}=`, 0, "CRITICAL"},
		{"azure-eventhub-connstr", `Endpoint=sb://[^;]+\.servicebus\.windows\.net/;SharedAccessKeyName=[^;]+;SharedAccessKey=[A-Za-z0-9+/]{43}=;EntityPath=[^\s]+`, 0, "CRITICAL"},
		{"azure-cosmosdb-key", `(?i)(?:cosmos[._-]?(?:db[._-]?)?(?:account[._-]?)?key|COSMOS_KEY)\s*[=:]\s*([A-Za-z0-9+/]{86}==)`, 1, "CRITICAL"},
		{"azure-search-admin-key", `(?i)(?:search[._-]?(?:admin[._-]?)?key|AZURE_SEARCH_KEY)\s*[=:]\s*([A-Za-z0-9]{32})`, 1, "CRITICAL"},
		{"azure-cognitive-key", `(?i)(?:cognitive[._-]?(?:services[._-]?)?key|AZURE_COGNITIVE_KEY)\s*[=:]\s*([a-f0-9]{32})`, 1, "CRITICAL"},
		{"azure-iot-hub-connstr", `HostName=[^;]+\.azure-devices\.net;SharedAccessKeyName=[^;]+;SharedAccessKey=[A-Za-z0-9+/]{43}=`, 0, "CRITICAL"},
		{"sharepoint-client-secret", `(?i)(?:SharePoint|SPO|M365)[._-]?(?:client[._-]?)?secret\s*[=:]\s*([A-Za-z0-9+/]{32,}={0,2})`, 1, "CRITICAL"},
		{"graph-api-client-secret", `(?i)(?:graph[._-]?(?:api[._-]?)?(?:client[._-]?)?secret|MS_GRAPH_SECRET)\s*[=:]\s*([A-Za-z0-9~._\-]{34,})`, 1, "CRITICAL"},
		{"teams-webhook", `https://[a-zA-Z0-9\-]+\.webhook\.office\.com/webhookb2/[A-Za-z0-9\-@]+/IncomingWebhook/[A-Za-z0-9]+/[A-Za-z0-9\-]+`, 0, "HIGH"},
		{"power-automate-shared-key", `(?i)(?:LogicApp|PowerAutomate|flow)[._-]?(?:shared[._-]?)?(?:access[._-]?)?key\s*[=:]\s*([A-Za-z0-9+/]{40,}={0,2})`, 1, "HIGH"},
		// Frontend / SaaS
		{"firebase-private-key", `(?i)(?:firebase|FIREBASE)[._-]?(?:admin[._-]?)?(?:private[._-]?)?key[._-]?(?:id)?\s*[=:]\s*([A-Za-z0-9]{40})`, 1, "CRITICAL"},
		{"mapbox-public-token", `pk\.eyJ[A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+`, 0, "LOW"},
		{"mapbox-secret-token", `sk\.eyJ[A-Za-z0-9\-_=]+\.[A-Za-z0-9\-_=]+`, 0, "HIGH"},
		{"sentry-dsn", `https://[a-f0-9]{32}@[a-z0-9\-]+\.ingest\.sentry\.io/[0-9]+`, 0, "MEDIUM"},
		{"contentful-pat", `CFPAT-[A-Za-z0-9\-_]{43}`, 0, "HIGH"},
		{"shopify-pat", `shpat_[A-Fa-f0-9]{32}`, 0, "CRITICAL"},
		{"shopify-shared-secret", `shpss_[A-Fa-f0-9]{32}`, 0, "CRITICAL"},
		{"shopify-custom-app", `shpca_[A-Fa-f0-9]{32}`, 0, "CRITICAL"},
		{"shopify-private-app", `shppa_[A-Fa-f0-9]{32}`, 0, "CRITICAL"},
		{"algolia-admin-key", `(?i)(?:algolia[._-]?(?:admin[._-]?)?(?:api[._-]?)?key|ALGOLIA_ADMIN_KEY)\s*[=:]["'\s]*([a-f0-9]{32})`, 1, "CRITICAL"},
		{"linear-api-key", `lin_api_[A-Za-z0-9]{40}`, 0, "HIGH"},
		{"postman-api-key", `PMAK-[A-Za-z0-9\-]{58,}`, 0, "HIGH"},
		{"planetscale-token", `pscale_tkn_[A-Za-z0-9\-_]{43}`, 0, "HIGH"},
		{"cloudflare-api-token", `(?i)(?:CF[._-]?API[._-]?TOKEN|CLOUDFLARE[._-]?(?:API[._-]?)?TOKEN)\s*[=:]\s*([A-Za-z0-9\-_]{40})`, 1, "HIGH"},
		{"cloudflare-api-key", `(?i)(?:CF[._-]?API[._-]?KEY|CLOUDFLARE[._-]?(?:API[._-]?)?KEY)\s*[=:]\s*([a-f0-9]{37})`, 1, "CRITICAL"},
		// Database
		{"db-postgres-url", `postgres(?:ql)?://[^:@\s]+:[^:@\s]+@[^/\s]+/\S+`, 0, "CRITICAL"},
		{"db-mysql-url", `mysql(?:2)?://[^:@\s]+:[^:@\s]+@[^/\s]+/\S+`, 0, "CRITICAL"},
		{"db-mongodb-url", `mongodb(?:\+srv)?://[^:@\s]+:[^:@\s]+@[^/\s]+(?:/\S*)?`, 0, "CRITICAL"},
		{"db-redis-url", `rediss?://(?:[^:@\s]+:)[^@\s]+@[^/\s]+(?:/\d+)?`, 0, "HIGH"},
		{"db-mssql-connstr", `(?i)(?:Server|Data Source)=[^;]+;[^;]*(?:Password|PWD)=([^;]+)`, 1, "CRITICAL"},
		{"db-elasticsearch-url", `https?://[^:@\s]+:[^:@\s]+@[^/\s]*(?:920[0-9]|930[0-9])[^\s]*`, 0, "HIGH"},
		{"db-amqp-url", `amqps?://[^:@\s]+:[^:@\s]+@[^/\s]+`, 0, "HIGH"},
		{"db-generic-password", `(?i)(?:db|database)[_\-\.]?(?:password|passwd|pwd)\s*[=:]\s*[^\s"']{8,}`, 0, "HIGH"},
		{"db-jdbc-url", `jdbc:[a-z0-9]+://[^:@\s]*:[^@\s]+@[^\s]+`, 0, "CRITICAL"},
		// Observability
		{"otel-endpoint-with-auth", `https?://[^:@\s]+:[^:@\s]+@[^\s]*(?:4317|4318|otlp|otel|opentelemetry)[^\s]*`, 0, "HIGH"},
		{"otel-exporter-headers", `(?i)OTEL_EXPORTER_OTLP_HEADERS\s*=\s*[^\n]*(?:api[_-]?key|authorization|x-honeycomb-team)=[A-Za-z0-9\-_+/=]{12,}`, 0, "HIGH"},
		{"honeycomb-api-key", `(?i)(?:x-honeycomb-team|honeycomb[._-]?(?:api[._-]?)?key)\s*[=:]\s*([A-Za-z0-9]{22,})`, 1, "HIGH"},
		{"datadog-api-key", `(?i)(?:DD_API_KEY|datadog[._-]?api[._-]?key)\s*[=:]\s*([a-f0-9]{32})`, 1, "HIGH"},
		{"newrelic-license-key", `(?i)(?:NEW_RELIC_LICENSE_KEY|newrelic[._-]?license[._-]?key)\s*[=:]\s*([A-Za-z0-9]{40})`, 1, "HIGH"},
		{"grafana-service-account", `glsa_[A-Za-z0-9]{32}_[A-Fa-f0-9]{8}`, 0, "HIGH"},
		{"lightstep-token", `(?i)(?:x-lightstep-access-token|lightstep[._-]?token)\s*[=:]\s*([A-Za-z0-9\-_]{20,})`, 1, "HIGH"},
		// HTTP Auth
		{"http-basic-auth-header", `(?i)(?:Authorization|auth)\s*[:=]\s*Basic\s+([A-Za-z0-9+/]{8,}={0,2})`, 1, "HIGH"},
		{"http-basic-auth-curl", `curl\s+[^\n]*(?:-u|--user)\s+([^:'\s"]+:[^@'\s"]+)`, 1, "HIGH"},
		{"http-basic-auth-env", `(?i)BASIC[_-]?AUTH\s*[=:]\s*([A-Za-z0-9+/]{8,}={0,2})`, 1, "HIGH"},
		{"http-bearer-header", `(?i)(?:Authorization|auth)\s*[:=]\s*Bearer\s+([A-Za-z0-9\-_=+/.]{16,})`, 1, "HIGH"},
		{"http-bearer-env", `(?i)BEARER[_-]?TOKEN\s*[=:]\s*([A-Za-z0-9\-_=+/.]{16,})`, 1, "HIGH"},
		{"http-bearer-curl", `(?i)curl\s+[^\n]*-H\s+"Authorization:\s*Bearer\s+([A-Za-z0-9\-_=+/.]{16,})"`, 1, "HIGH"},
		{"http-insecure-url", `http://[a-zA-Z0-9\-._~:/?#@!$&()*+,;=%]{4,}`, 0, "WARNING"},
		{"http-auth-over-http", `(?i)http://[^:@\s]+:[^:@\s]+@[^\s]+`, 0, "CRITICAL"},
		// Infrastructure / CI
		{"vault-service-token", `hvs\.[A-Za-z0-9_-]{24,}`, 0, "CRITICAL"},
		{"vault-batch-token", `hvb\.[A-Za-z0-9_-]{24,}`, 0, "CRITICAL"},
		{"terraform-cloud-token", `TFC-[A-Za-z0-9]{14,}`, 0, "CRITICAL"},
		{"digitalocean-pat", `dop_v1_[a-f0-9]{64}`, 0, "CRITICAL"},
		{"circleci-token", `ccipat_[A-Za-z0-9]{40,}`, 0, "HIGH"},
		// Email providers
		{"resend-api-key", `re_[A-Za-z0-9_-]{24,}`, 0, "HIGH"},
		{"mailgun-api-key", `\bkey-[a-f0-9]{32}\b`, 0, "HIGH"},
		{"postmark-server-token", `(?i)(?:POSTMARK[._-]?(?:SERVER[._-]?)?(?:API[._-]?)?TOKEN|X-Postmark-Server-Token)\s*[=:]\s*([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`, 1, "HIGH"},
		// Database / BaaS
		{"supabase-pat", `sbp_[a-f0-9]{40}`, 0, "CRITICAL"},
		{"supabase-service-role", `(?i)SUPABASE[._-]?(?:SERVICE[._-]?ROLE[._-]?)?KEY\s*[=:]\s*(eyJ[A-Za-z0-9_-]{30,}\.[A-Za-z0-9_-]{30,}\.[A-Za-z0-9_-]{27,})`, 1, "CRITICAL"},
		// AI / Embeddings
		{"pinecone-api-key", `pcsk_[A-Za-z0-9_]{40,}`, 0, "CRITICAL"},
		{"elevenlabs-api-key", `(?i)(?:ELEVENLABS[._-]?(?:API[._-]?)?KEY|XI_API_KEY)\s*[=:]\s*([a-f0-9]{32})`, 1, "HIGH"},
		{"openai-api-key-env", `(?i)(?:OPENAI[._-]?API[._-]?KEY)\s*[=:]\s*(sk-(?:proj-|svcacct-)?[A-Za-z0-9\-_]{20,})`, 1, "CRITICAL"},
		{"anthropic-api-key-env", `(?i)(?:ANTHROPIC[._-]?API[._-]?KEY)\s*[=:]\s*(sk-ant-(?:api03-)?[A-Za-z0-9\-_]{16,})`, 1, "CRITICAL"},
		// Python / LLM
		{"python-llm-key-assignment", `(?i)\b(?:openai|anthropic|azure_openai|mistral|cohere|groq|together|huggingface|hf|replicate|perplexity|xai|langsmith|pinecone|weaviate)[._-]?(?:api[._-]?)?(?:key|token|secret)\b\s*=\s*["'][^"'\n]{16,}["']`, 0, "CRITICAL"},
		{"python-dotenv-llm-key", `(?im)^\s*(?:OPENAI|ANTHROPIC|AZURE_OPENAI|MISTRAL|COHERE|GROQ|TOGETHER|HUGGINGFACE|HF|REPLICATE|PERPLEXITY|XAI|LANGSMITH|PINECONE|WEAVIATE)[A-Z0-9_]*_(?:API_)?(?:KEY|TOKEN|SECRET)\s*=\s*[^\s#]{16,}\s*$`, 0, "CRITICAL"},
		{"langsmith-api-key-env", `(?i)\b(?:LANGSMITH|LANGCHAIN)[A-Z0-9_]*_(?:API_)?KEY\s*[=:]\s*([A-Za-z0-9_\-]{20,})`, 1, "CRITICAL"},
		{"python-os-environ-secret", `(?i)\bos\.environ\[\s*["'][A-Z0-9_]*(?:API_)?(?:KEY|TOKEN|SECRET)[A-Z0-9_]*["']\s*\]\s*=\s*["'][^"'\n]{12,}["']`, 0, "HIGH"},
		{"python-openai-client-inline-key", `(?i)\bOpenAI\s*\(\s*api_key\s*=\s*["'][^"'\n]{16,}["']`, 0, "CRITICAL"},
		{"python-anthropic-client-inline-key", `(?i)\bAnthropic\s*\(\s*api_key\s*=\s*["'][^"'\n]{16,}["']`, 0, "CRITICAL"},
		{"vertex-private-key-json", `"private_key"\s*:\s*"-----BEGIN PRIVATE KEY-----`, 0, "CRITICAL"},
		{"aws-bedrock-access-key-env", `(?i)\bAWS_ACCESS_KEY_ID\s*[=:]\s*((?:AKIA|ASIA)[A-Z0-9]{16})`, 1, "CRITICAL"},
		{"aws-bedrock-secret-key-env", `(?i)\bAWS_SECRET_ACCESS_KEY\s*[=:]\s*([A-Za-z0-9/+=]{40})`, 1, "CRITICAL"},
		{"streamlit-secrets-llm-key", `(?i)\b(?:openai|anthropic|langsmith|cohere|mistral|groq|together|huggingface|replicate|perplexity|xai)[._-]?(?:api[._-]?)?(?:key|token|secret)\s*=\s*["'][^"'\n]{16,}["']`, 0, "CRITICAL"},
		// Communication
		{"slack-signing-secret", `(?i)(?:SLACK_SIGNING_SECRET|slack[._-]?signing[._-]?secret)\s*[=:]\s*([a-f0-9]{32})`, 1, "HIGH"},
		{"slack-app-token", `xapp-[0-9]-[A-Z0-9]+-[0-9]+-[a-f0-9]{64}`, 0, "HIGH"},
		{"twilio-auth-token", `(?i)TWILIO[._-]?AUTH[._-]?TOKEN\s*[=:]\s*([a-f0-9]{32})`, 1, "CRITICAL"},
		{"discord-bot-token", `(?i)(?:discord[._-]?(?:bot[._-]?)?token|DISCORD_TOKEN)\s*[=:]\s*([A-Za-z0-9_-]{24,26}\.[A-Za-z0-9_-]{6}\.[A-Za-z0-9_-]{27,})`, 1, "HIGH"},
		{"telegram-bot-token", `[0-9]{8,12}:[A-Za-z0-9_-]{35}`, 0, "HIGH"},
		{"notion-integration-token", `secret_[A-Za-z0-9]{43}`, 0, "HIGH"},
		{"notion-oauth-token", `ntn_[A-Za-z0-9]{40,}`, 0, "HIGH"},
		// GCP
		{"gcp-service-account", `"type"\s*:\s*"service_account"`, 0, "CRITICAL"},
	}

	for _, r := range rules {
		re, err := regexp.Compile(r.pat)
		if err != nil {
			continue
		}
		secretRules = append(secretRules, secretRule{
			id: r.id, pattern: re, secretGroup: r.group,
			confidence: sev[r.sev],
		})
	}
}

func detectSecrets(text string) []Finding {
	var out []Finding
	for _, rule := range secretRules {
		for _, m := range rule.pattern.FindAllStringSubmatchIndex(text, -1) {
			si, ei := rule.secretGroup*2, rule.secretGroup*2+1
			if ei >= len(m) || m[si] < 0 {
				continue
			}
			out = append(out, Finding{
				Type: PiiSecret, Start: m[si], End: m[ei],
				Text: text[m[si]:m[ei]], Confidence: rule.confidence, RuleID: rule.id,
			})
		}
	}
	return out
}
