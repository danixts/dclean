package domain

var ProjectCategories = []Category{
	{
		Name:    "Node Modules",
		Color:   "#68a063",
		Targets: []string{"node_modules"},
	},
	{
		Name:    "Package Manager Store",
		Color:   "#cb3837",
		Targets: []string{".npm", ".pnpm-store", ".yarn", ".bun"},
	},
	{
		Name:    "Next.js",
		Color:   "#ffffff",
		Targets: []string{".next"},
	},
	{
		Name:    "Turborepo",
		Color:   "#0096ff",
		Targets: []string{".turbo"},
	},
	{
		Name:    "Build Output",
		Color:   "#f5a623",
		Targets: []string{"dist", "build", ".output", ".nuxt", ".svelte-kit", ".angular"},
	},
	{
		Name:    "Dev Cache",
		Color:   "#888888",
		Targets: []string{".parcel-cache", ".vite", ".eslintcache", ".temp"},
	},
	{
		Name:    "Test Coverage",
		Color:   "#a855f7",
		Targets: []string{"coverage", ".nyc_output"},
	},
	{
		Name:    "Python Cache",
		Color:   "#3776ab",
		Targets: []string{"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache"},
	},
}

var HomeCacheCategories = []Category{
	{
		Name:    "Go Build Cache",
		Color:   "#00add8",
		Targets: []string{"go-build", "gopls", "goimports", "golangci-lint"},
	},
	{
		Name:    "IDE Cache",
		Color:   "#fc801d",
		Targets: []string{"JetBrains", "cursor-compile-cache"},
	},
	{
		Name:    "Package Manager Cache",
		Color:   "#cb3837",
		Targets: []string{"pip", "pnpm", "yarn", "uv", "npm", "turbo"},
	},
	{
		Name:    "Browser Cache",
		Color:   "#4285f4",
		Targets: []string{"google-chrome", "mozilla", "BraveSoftware", "microsoft-edge"},
	},
	{
		Name:    "Dev Tools Cache",
		Color:   "#38bdf8",
		Targets: []string{"typescript", "eslint", "prettier", "ms-playwright", "helm", "opencode"},
	},
	{
		Name:    "System Cache",
		Color:   "#888888",
		Targets: []string{"thumbnails", "tracker3", "fontconfig"},
	},
}

const (
	SnapOldRevisionsCategory = "Snap Old Revisions"
	SnapOldRevisionsColor    = "#ff6600"
	SnapCacheCategory        = "Snap Cache"
	SnapCacheColor           = "#ff9933"
	SnapCacheMinSize         = 1024 * 1024
)
