package binarycache

const (
	DefaultCacheName              = ".genmcp"
	DefaultBinaryPrefix           = "genmcp-server"
	DefaultGitHubReleasesURL      = "https://github.com/genmcp/gen-mcp/releases/download"
	DefaultGitHubAPIURL           = "https://api.github.com/repos/genmcp/gen-mcp/releases/latest"
	DefaultSigstoreIdentityRegexp = "https://github.com/genmcp/gen-mcp/.*"
	DefaultSigstoreOIDCIssuer     = "https://token.actions.githubusercontent.com"
	DefaultVerbose                = false
)

type Config struct {
	CacheName              string // e.g. ".genmcp"
	BinaryPrefix           string // e.g. "genmcp-server"
	GitHubReleasesURL      string
	GitHubAPIURL           string
	SigstoreIdentityRegexp string
	SigstoreOIDCIssuer     string
	Verbose                bool // whether to log out verbose outputs
}

func DefaultConfig() *Config {
	return &Config{
		CacheName:              DefaultCacheName,
		BinaryPrefix:           DefaultBinaryPrefix,
		GitHubReleasesURL:      DefaultGitHubReleasesURL,
		GitHubAPIURL:           DefaultGitHubAPIURL,
		SigstoreIdentityRegexp: DefaultSigstoreIdentityRegexp,
		SigstoreOIDCIssuer:     DefaultSigstoreOIDCIssuer,
		Verbose:                DefaultVerbose,
	}
}

func (cfg *Config) GetCacheName() string {
	if cfg == nil || cfg.CacheName == "" {
		return DefaultCacheName
	}

	return cfg.CacheName
}

// GetSanitizedCacheName returns the cache name with path separators replaced,
// making it safe for use in temp directory prefixes and filenames.
func (cfg *Config) GetSanitizedCacheName() string {
	name := cfg.GetCacheName()
	// Replace path separators with underscores to make it filesystem-safe
	result := make([]byte, len(name))
	for i := 0; i < len(name); i++ {
		if name[i] == '/' || name[i] == '\\' {
			result[i] = '_'
		} else {
			result[i] = name[i]
		}
	}
	return string(result)
}

func (cfg *Config) GetBinaryPrefix() string {
	if cfg == nil || cfg.BinaryPrefix == "" {
		return DefaultBinaryPrefix
	}

	return cfg.BinaryPrefix
}

func (cfg *Config) GetGitHubReleasesURL() string {
	if cfg == nil || cfg.GitHubReleasesURL == "" {
		return DefaultGitHubReleasesURL
	}

	return cfg.GitHubReleasesURL
}

func (cfg *Config) GetGitHubAPIURL() string {
	if cfg == nil || cfg.GitHubAPIURL == "" {
		return DefaultGitHubAPIURL
	}

	return cfg.GitHubAPIURL
}

func (cfg *Config) GetSigstoreIdentityRegexp() string {
	if cfg == nil || cfg.SigstoreIdentityRegexp == "" {
		return DefaultSigstoreIdentityRegexp
	}

	return cfg.SigstoreIdentityRegexp
}

func (cfg *Config) GetSigstoreOIDCIssuer() string {
	if cfg == nil || cfg.SigstoreOIDCIssuer == "" {
		return DefaultSigstoreOIDCIssuer
	}

	return cfg.SigstoreOIDCIssuer
}

func (cfg *Config) GetVerbose() bool {
	if cfg == nil {
		return DefaultVerbose
	}

	return cfg.Verbose
}
