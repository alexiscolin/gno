package gnoweb

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/xid"

	"github.com/gnolang/gno/gno.land/pkg/integration"
	"github.com/gnolang/gno/gnovm/pkg/gnoenv"
	"github.com/gnolang/gno/tm2/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ok         = http.StatusOK
	found      = http.StatusFound
	notFound   = http.StatusNotFound
	badRequest = http.StatusBadRequest
)

func TestRoutes(t *testing.T) {
	var (
		uuid1   = xid.New()
		uuid2   = xid.New()
		aliases = map[string]AliasTarget{
			"/test1": {Value: "/r/gnoland/users/v1", Kind: GnowebPath},
			"/test2": {Value: uuid1.String(), Kind: StaticMarkdown},
			"/test3": {Value: "/r/not/found", Kind: GnowebPath},
			"/test4": {Value: uuid2.String(), Kind: StaticMarkdown},
		}
		routes = []struct {
			route     string
			status    int
			substring string
		}{
			{"/", ok, "Welcome"}, // Check if / returns 200 (OK) and contains "Welcome".
			{"/about", ok, "blockchain"},
			{"/r/gnoland/blog", ok, ""}, // Any content
			{"/r/gnoland/blog$help", ok, "AdminSetAdminAddr"},
			{"/r/gnoland/blog/", ok, "admin.gno"},
			{"/r/gnoland/blog/admin.gno", ok, ">func<"},
			{"/r/gnoland/blog$help&func=Render", ok, "Render(path)"},
			{"/r/gnoland/blog$help&func=Render&path=foo/bar", ok, `value="foo/bar"`},
			// {"/r/gnoland/blog$help&func=NonExisting", ok, "NonExisting not found"}, // XXX(TODO)
			{"/r/gnoland/users/v1:archives", ok, "Address"},
			{"/r/gnoland/users/v1", ok, "registry"},
			{"/r/gnoland/users/v1/users.gno", ok, "reValidUsername"},
			{"/r/demo/deep/very/deep", ok, "it works!"},
			{"/r/demo/deep/very/deep?arg1=val1&arg2=val2", ok, "hi ?arg1=val1&amp;arg2=val2"},
			{"/r/demo/deep/very/deep:bob", ok, "hi bob"},
			{"/r/demo/deep/very/deep:bob?arg1=val1&arg2=val2", ok, "hi bob?arg1=val1&amp;arg2=val2"},
			{"/r/demo/deep/very/deep$help", ok, "Render"},
			{"/r/demo/deep/very/deep/", ok, "render.gno"},
			{"/r/demo/deep/very/deep/render.gno", ok, ">package<"},
			{"/contribute", ok, "Game of Realms"},
			{"/game-of-realms", found, "/contribute"},
			{"/gor", found, "/contribute"},
			{"/blog", found, "/r/gnoland/blog"},
			{"/r/docs/optional_render", http.StatusOK, "gnomod.toml"},
			{"/r/not/found/", notFound, ""},
			{"/z/bad/request", badRequest, ""}, // not realm or pure
			{"/아스키문자가아닌경로", notFound, ""},
			{"/%ED%85%8C%EC%8A%A4%ED%8A%B8", notFound, ""},
			{"/グノー", notFound, ""},
			{"/\u269B\uFE0F", notFound, ""}, // Unicode
			{"/p/demo/flow/LICENSE", ok, "BSD 3-Clause"},
			// Test assets
			{"/public/styles.css", ok, ""},
			{"/public/js/index.js", ok, ""},
			{"/public/_chroma/style.css", ok, ""},
			{"/public/imgs/gnoland.svg", ok, ""},
			// Test Toc
			{"/", ok, `href="#learn-about-gnoland"`},
			// Test aliased path and static file
			{"/test1", ok, "registry"},     // Alias "/test1" points to "/r/gnoland/users/v1"
			{"/test2", ok, uuid1.String()}, // Alias "/test2" points to static file containing an uuid
			{"/test3", notFound, ""},       // Alias "/test3" points to "/r/not/found" which doesn't exist
			{"/test4", ok, uuid2.String()}, // Alias "/test2_b" points to another static file containing an uuid
			{"/test123", badRequest, ""},   // Alias "/test123" doesn't exist, points to "" which is not valid
		}
	)

	// Setup the logger and node
	logger := log.NewTestingLogger(t)
	rootdir := gnoenv.RootDir()
	genesis := integration.LoadDefaultGenesisTXsFile(t, "tendermint_test", rootdir)
	config, _ := integration.TestingNodeConfig(t, rootdir, genesis...)
	node, remoteAddr := integration.TestingInMemoryNode(t, logger, config)
	defer node.Stop()

	// Initialize the router with the current node's remote address
	cfg := NewDefaultAppConfig()
	cfg.NodeRemote = remoteAddr
	for alias, target := range aliases {
		cfg.Aliases[alias] = target
	}
	router, err := NewRouter(logger, cfg)
	require.NoError(t, err)

	for _, r := range routes {
		t.Run(fmt.Sprintf("test route %s", r.route), func(t *testing.T) {
			t.Logf("input: %q", r.route)
			request := httptest.NewRequest(http.MethodGet, r.route, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			assert.Equal(t, r.status, response.Code)
			assert.Contains(t, response.Body.String(), r.substring)
		})
	}
}

func TestAnalytics(t *testing.T) {
	routes := []string{
		// Special realms
		"/", // Home
		"/about",
		"/start",

		// Redirects
		"/game-of-realms",
		"/getting-started",
		"/blog",
		"/boards",

		// Realm, source, help page
		"/r/gnoland/blog",
		"/r/gnoland/blog/admin.gno",
		"/r/gnoland/users/v1",
		"/r/gnoland/blog$help",

		// Special pages
		"/404-not-found",
	}

	rootdir := gnoenv.RootDir()
	genesis := integration.LoadDefaultGenesisTXsFile(t, "tendermint_test", rootdir)
	config, _ := integration.TestingNodeConfig(t, rootdir, genesis...)
	node, remoteAddr := integration.TestingInMemoryNode(t, log.NewTestingLogger(t), config)
	defer node.Stop()

	t.Run("enabled", func(t *testing.T) {
		for _, route := range routes {
			t.Run(route, func(t *testing.T) {
				cfg := NewDefaultAppConfig()
				cfg.NodeRemote = remoteAddr
				cfg.Analytics = true
				logger := log.NewTestingLogger(t)

				router, err := NewRouter(logger, cfg)
				require.NoError(t, err)

				request := httptest.NewRequest(http.MethodGet, route, nil)
				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				assert.Contains(t, response.Body.String(), "sa.gno.services")
			})
		}
	})
	t.Run("disabled", func(t *testing.T) {
		for _, route := range routes {
			t.Run(route, func(t *testing.T) {
				cfg := NewDefaultAppConfig()
				cfg.NodeRemote = remoteAddr
				cfg.Analytics = false
				logger := log.NewTestingLogger(t)
				router, err := NewRouter(logger, cfg)
				require.NoError(t, err)

				request := httptest.NewRequest(http.MethodGet, route, nil)
				response := httptest.NewRecorder()

				router.ServeHTTP(response, request)

				assert.NotContains(t, response.Body.String(), "sa.gno.services")
			})
		}
	})
}
