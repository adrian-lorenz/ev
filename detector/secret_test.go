package detector

import "testing"

// Secrets are split across string literals so GitHub secret scanning
// does not flag this test file as containing live credentials.
var (
	testAWSKey     = "key=" + "AKIA" + "IOSFODNN7EXAMPLE"
	testGHPat      = "token: " + "ghp_" + "ABCDeFgHiJkLmNoPqRsTuV" + "wXyZ1234567890"
	testGLPat      = "glpat-" + "ABCDEFGHIJ" + "KLMNOPQRST"
	testPGURL      = "postgres://" + "user:password@localhost:5432/mydb"
	testMongoURL   = "mongodb://" + "user:pass@cluster.mongodb.net/db"
	testJWT        = "eyJhbGciOiJIUzI1NiJ9." + "eyJzdWIiOiIxMjM0NTY3ODkwIn0." + "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	testGenericSec = `password = "mySecretPass` + `word123"`
	testPyOAIKey   = `client = OpenAI(api_key="sk-test` + `key1234567890abcdefghij")`
)

func TestDetectSecrets(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantN      int
		wantRuleID string // rule that must fire (empty = any)
		wantText   string // expected secret value (empty = any)
	}{
		{
			"aws access key",
			testAWSKey,
			1, "aws-access-key", "AKIA" + "IOSFODNN7EXAMPLE",
		},
		{
			"github pat",
			testGHPat,
			1, "github-pat", "",
		},
		{
			"gitlab pat",
			testGLPat,
			1, "gitlab-pat", "",
		},
		{
			"anthropic api key",
			"sk-ant-" + "api03-" + repeatStr("A", 32),
			1, "anthropic-api-key", "",
		},
		{
			"openai sk-proj",
			"sk-proj-" + repeatStr("A", 50),
			1, "openai-api-key-new", "",
		},
		{
			"huggingface token",
			"hf_" + repeatStr("a", 32),
			1, "huggingface-token", "",
		},
		{
			"groq api key",
			"gsk_" + repeatStr("A", 52),
			1, "groq-api-key", "",
		},
		{
			"postgres url",
			testPGURL,
			1, "db-postgres-url", "",
		},
		{
			"mongodb url",
			testMongoURL,
			1, "db-mongodb-url", "",
		},
		{
			"pem private key header",
			"-----BEGIN RSA " + "PRIVATE KEY-----",
			1, "private-key-header", "",
		},
		{
			"jwt token",
			testJWT,
			1, "jwt-token", "",
		},
		{
			"generic secret assignment",
			testGenericSec,
			1, "generic-secret", "mySecretPass" + "word123",
		},
		{
			"python inline openai key",
			testPyOAIKey,
			1, "python-openai-client-inline-key", "",
		},
		{
			"gcp service account",
			`{"type": "service_account", "project_id": "my-project"}`,
			1, "gcp-service-account", "",
		},
		{
			"only secret value redacted not key name",
			"ANTHROPIC_API_KEY=" + "sk-ant-" + "api03-" + repeatStr("B", 32),
			1, "anthropic-api-key-env", "",
		},
		{
			"no match plain text",
			"Hallo, das ist ein normaler Text ohne Secrets.",
			0, "", "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectSecrets(tt.input)
			if len(got) < tt.wantN {
				t.Fatalf("got %d findings, want at least %d — %v", len(got), tt.wantN, got)
			}
			if tt.wantN == 0 {
				return
			}
			if tt.wantRuleID != "" {
				found := false
				for _, f := range got {
					if f.RuleID == tt.wantRuleID {
						found = true
						if tt.wantText != "" && f.Text != tt.wantText {
							t.Errorf("rule %q: text = %q, want %q", tt.wantRuleID, f.Text, tt.wantText)
						}
						break
					}
				}
				if !found {
					t.Errorf("rule %q not fired — got rules: %v", tt.wantRuleID, ruleIDs(got))
				}
			}
			for _, f := range got {
				if f.Type != PiiSecret {
					t.Errorf("type = %v, want SECRET", f.Type)
				}
				if f.Confidence <= 0 {
					t.Errorf("confidence = %v, want > 0", f.Confidence)
				}
			}
		})
	}
}

func ruleIDs(findings []Finding) []string {
	ids := make([]string, len(findings))
	for i, f := range findings {
		ids[i] = f.RuleID
	}
	return ids
}
