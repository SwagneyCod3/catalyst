package test

import (
	"context"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/arangodb/go-driver"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi"
	"golang.org/x/oauth2"

	"github.com/SecurityBrewery/catalyst"
	"github.com/SecurityBrewery/catalyst/auth"
	"github.com/SecurityBrewery/catalyst/bus"
	"github.com/SecurityBrewery/catalyst/database"
	"github.com/SecurityBrewery/catalyst/database/busdb"
	"github.com/SecurityBrewery/catalyst/generated/api"
	"github.com/SecurityBrewery/catalyst/generated/model"
	"github.com/SecurityBrewery/catalyst/hooks"
	"github.com/SecurityBrewery/catalyst/index"
	"github.com/SecurityBrewery/catalyst/service"
	"github.com/SecurityBrewery/catalyst/storage"
)

func Context() context.Context {
	return busdb.UserContext(context.Background(), Bob)
}

func Config(ctx context.Context) (*catalyst.Config, error) {
	config := &catalyst.Config{
		InitialAPIKey: "test",
		IndexPath:     "index.bleve",
		Network:       "catalyst",
		DB: &database.Config{
			Host:     "http://localhost:8529",
			User:     "root",
			Password: "foobar",
		},
		Storage: &storage.Config{
			Host:     "http://localhost:9000",
			User:     "minio",
			Password: "minio123",
		},
		Secret: []byte("4ef5b29539b70233dd40c02a1799d25079595565e05a193b09da2c3e60ada1cd"),
		Auth: &auth.Config{
			SimpleAuthEnable: true,
			APIKeyAuthEnable: true,
			OIDCAuthEnable:   true,
			OIDCIssuer:       "http://localhost:9002/auth/realms/catalyst",
			OAuth2: &oauth2.Config{
				ClientID:     "catalyst",
				ClientSecret: "13d4a081-7395-4f71-a911-bc098d8d3c45",
				RedirectURL:  "http://localhost:8002/callback",
				Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
			},
			// OIDCClaimUsername: "",
			// OIDCClaimEmail:    "",
			// OIDCClaimName:     "",
			// AuthBlockNew:      false,
			// AuthDefaultRoles:  nil,
		},
	}

	if err := config.Auth.Load(ctx); err != nil {
		return nil, err
	}

	return config, nil
}

func Index(t *testing.T) (*index.Index, func(), error) {
	t.Helper()

	dir, err := os.MkdirTemp("", "catalyst-test-"+cleanName(t))
	if err != nil {
		return nil, nil, err
	}

	catalystIndex, err := index.New(path.Join(dir, "index.bleve"))
	if err != nil {
		return nil, nil, err
	}

	return catalystIndex, func() { catalystIndex.Close(); os.RemoveAll(dir) }, nil
}

func Bus(t *testing.T) (context.Context, *catalyst.Config, *bus.Bus, error) {
	t.Helper()

	ctx := Context()

	config, err := Config(ctx)
	if err != nil {
		t.Fatal(err)
	}

	catalystBus := bus.New()

	return ctx, config, catalystBus, err
}

func DB(t *testing.T) (context.Context, *catalyst.Config, *bus.Bus, *index.Index, *storage.Storage, *database.Database, func(), error) {
	t.Helper()

	ctx, config, rbus, err := Bus(t)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	catalystStorage, err := storage.New(config.Storage)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	catalystIndex, cleanup, err := Index(t)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	c := config.DB
	c.Name = cleanName(t)
	db, err := database.New(ctx, catalystIndex, rbus, &hooks.Hooks{
		DatabaseAfterConnectFuncs: []func(ctx context.Context, client driver.Client, name string){Clear},
	}, c)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	_, err = db.JobCreate(ctx, "b81c2366-ea37-43d2-b61b-03afdc21d985", &model.JobForm{
		Automation: "hash.sha1",
		Payload:    "test",
		Origin:     nil,
	})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	return ctx, config, rbus, catalystIndex, catalystStorage, db, func() {
		err := db.Remove(context.Background())
		if err != nil {
			log.Println(err)
		}
		cleanup()
	}, err
}

func Service(t *testing.T) (context.Context, *catalyst.Config, *bus.Bus, *index.Index, *storage.Storage, *database.Database, *service.Service, func(), error) {
	t.Helper()

	ctx, config, rbus, catalystIndex, catalystStorage, db, cleanup, err := DB(t)
	if err != nil {
		t.Fatal(err)
	}

	catalystService, err := service.New(rbus, db, catalystStorage, "0.0.0-test")
	if err != nil {
		t.Fatal(err)
	}

	return ctx, config, rbus, catalystIndex, catalystStorage, db, catalystService, cleanup, err
}

func Server(t *testing.T) (context.Context, *catalyst.Config, *bus.Bus, *index.Index, *storage.Storage, *database.Database, *service.Service, chi.Router, func(), error) {
	t.Helper()

	ctx, config, rbus, catalystIndex, catalystStorage, db, catalystService, cleanup, err := Service(t)
	if err != nil {
		t.Fatal(err)
	}

	catalystServer := api.NewServer(catalystService, func(s []string) func(http.Handler) http.Handler {
		return func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handler.ServeHTTP(w, busdb.SetContext(r, Bob))
			})
		}
	})

	return ctx, config, rbus, catalystIndex, catalystStorage, db, catalystService, catalystServer, cleanup, err
}

func Catalyst(t *testing.T) (context.Context, *catalyst.Config, *catalyst.Server, error) {
	t.Helper()

	ctx := Context()

	config, err := Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	config.DB.Name = cleanName(t)
	config.IndexPath = cleanName(t) + ".bleve"

	c, err := catalyst.New(&hooks.Hooks{
		DatabaseAfterConnectFuncs: []func(ctx context.Context, client driver.Client, name string){Clear},
	}, config)

	return ctx, config, c, err
}

func cleanName(t *testing.T) string {
	t.Helper()

	name := t.Name()
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "/", "_")

	return strings.ReplaceAll(name, "#", "_")
}

func Clear(ctx context.Context, client driver.Client, name string) {
	if exists, _ := client.DatabaseExists(ctx, name); exists {
		if db, err := client.Database(ctx, name); err == nil {
			if exists, _ = db.GraphExists(ctx, database.TicketArtifactsGraphName); exists {
				if g, err := db.Graph(ctx, database.TicketArtifactsGraphName); err == nil {
					if err := g.Remove(ctx); err != nil {
						log.Println(err)
					}
				}
			}
			if err := db.Remove(ctx); err != nil {
				log.Println(err)
			}
		}
	}
}
