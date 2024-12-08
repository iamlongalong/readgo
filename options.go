package readgo

import "time"

// AnalyzerOptions configures the behavior of the analyzer
type AnalyzerOptions struct {
	// WorkDir is the working directory for the analyzer
	WorkDir string

	// CacheTTL is the time-to-live for cached entries
	// If zero, caching is disabled
	CacheTTL time.Duration

	// MaxCacheSize is the maximum number of entries in each cache type
	// If zero, no limit is applied
	MaxCacheSize int

	// AnalysisTimeout is the timeout for analysis operations
	// If zero, no timeout is applied
	AnalysisTimeout time.Duration

	// EnableConcurrentAnalysis enables concurrent analysis for large projects
	EnableConcurrentAnalysis bool

	// MaxConcurrentAnalysis is the maximum number of concurrent analyses
	// If zero, defaults to runtime.NumCPU()
	MaxConcurrentAnalysis int
}

// DefaultOptions returns the default analyzer options
func DefaultOptions() *AnalyzerOptions {
	return &AnalyzerOptions{
		WorkDir:                  ".",
		CacheTTL:                 5 * time.Minute,
		MaxCacheSize:             1000,
		AnalysisTimeout:          30 * time.Second,
		EnableConcurrentAnalysis: true,
		MaxConcurrentAnalysis:    0, // Will use runtime.NumCPU()
	}
}

// Option is a function that configures AnalyzerOptions
type Option func(*AnalyzerOptions)

// WithWorkDir sets the working directory
func WithWorkDir(dir string) Option {
	return func(o *AnalyzerOptions) {
		o.WorkDir = dir
	}
}

// WithCacheTTL sets the cache TTL
func WithCacheTTL(ttl time.Duration) Option {
	return func(o *AnalyzerOptions) {
		o.CacheTTL = ttl
	}
}

// WithMaxCacheSize sets the maximum cache size
func WithMaxCacheSize(size int) Option {
	return func(o *AnalyzerOptions) {
		o.MaxCacheSize = size
	}
}

// WithAnalysisTimeout sets the analysis timeout
func WithAnalysisTimeout(timeout time.Duration) Option {
	return func(o *AnalyzerOptions) {
		o.AnalysisTimeout = timeout
	}
}

// WithConcurrentAnalysis enables or disables concurrent analysis
func WithConcurrentAnalysis(enable bool) Option {
	return func(o *AnalyzerOptions) {
		o.EnableConcurrentAnalysis = enable
	}
}

// WithMaxConcurrentAnalysis sets the maximum number of concurrent analyses
func WithMaxConcurrentAnalysis(max int) Option {
	return func(o *AnalyzerOptions) {
		o.MaxConcurrentAnalysis = max
	}
}
