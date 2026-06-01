package idx

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mcp/apiclient"
	"github.com/quantyralabs/idx-api/internal/mcp/ratelimit"
	"github.com/quantyralabs/idx-api/internal/repository"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	"github.com/quantyralabs/idx-api/internal/service/gis"
	"github.com/quantyralabs/idx-api/internal/service/search"
)

// Server registers API-parity MCP tools on a shared MCPServer instance.
type Server struct {
	cfg          config.Config
	db           *repository.DB
	keyRepo      *repository.MCPKeyRepo
	postgis      *search.PostgisSearch
	apiClient    *apiclient.Client
	domainSlug   string
	autocomplete *gis.AutocompleteService
	rateLimiter  *ratelimit.Limiter
}

// NewServer wires idx MCP tools.
func NewServer(
	cfg config.Config,
	keyRepo *repository.MCPKeyRepo,
	db *repository.DB,
	apiClient *apiclient.Client,
	domainSlug string,
	rateLimiter *ratelimit.Limiter,
) *Server {
	return &Server{
		cfg:          cfg,
		db:           db,
		keyRepo:      keyRepo,
		postgis:      search.NewPostgisSearch(db),
		apiClient:    apiClient,
		domainSlug:   domainSlug,
		autocomplete: gis.NewAutocompleteService(gisrepo.New(db)),
		rateLimiter:  rateLimiter,
	}
}

// RegisterTools adds API parity tools to the MCP server.
func (s *Server) RegisterTools(mcpServer *server.MCPServer) {
	if s == nil || mcpServer == nil {
		return
	}
	s.registerSearchTools(mcpServer)
	s.registerListingTools(mcpServer)
	s.registerGISTools(mcpServer)
	s.registerProxyTools(mcpServer)
}
