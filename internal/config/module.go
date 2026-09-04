package config

import (
	"github.com/bit-issues/backend/internal/attachments"
	"github.com/bit-issues/backend/internal/jwt"
	"github.com/bit-issues/backend/internal/oauth"
	"github.com/bit-issues/backend/internal/storage"
	"github.com/bit-issues/backend/internal/webauthn"
	"github.com/bit-issues/backend/internal/webhooks"
	"github.com/go-core-fx/cachefx"
	"github.com/go-core-fx/fiberfx"
	"github.com/go-core-fx/fiberfx/openapi"
	"github.com/go-core-fx/sqlfx"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"config",
		fx.Provide(New, fx.Private),
		fx.Provide(
			func(cfg Config) fiberfx.Config {
				return fiberfx.Config{
					Address:     cfg.HTTP.Address,
					ProxyHeader: cfg.HTTP.ProxyHeader,
					Proxies:     cfg.HTTP.Proxies,
				}
			},
			func(cfg Config) openapi.Config {
				return openapi.Config{
					Enabled:    cfg.HTTP.OpenAPI.Enabled,
					PublicHost: cfg.HTTP.OpenAPI.PublicHost,
					PublicPath: cfg.HTTP.OpenAPI.PublicPath,
				}
			},
			func(cfg Config) sqlfx.Config {
				return sqlfx.Config{
					URL:             cfg.Database.URL,
					ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
					ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
					MaxOpenConns:    cfg.Database.MaxOpenConns,
					MaxIdleConns:    cfg.Database.MaxIdleConns,
				}
			},
			func(cfg Config) cachefx.Config {
				return cachefx.Config{
					URL: cfg.Cache.URL,
				}
			},
		),
		fx.Provide(
			func(cfg Config) jwt.Config {
				return jwt.Config{
					Secret:     cfg.JWT.Secret,
					AccessTTL:  cfg.JWT.AccessTTL,
					RefreshTTL: cfg.JWT.RefreshTTL,
					Issuer:     cfg.JWT.Issuer,
				}
			},
			func(cfg Config) storage.Config {
				return storage.Config{
					URL:      cfg.Storage.URL,
					LinksTTL: cfg.Storage.LinksTTL,
				}
			},
			func(cfg Config) attachments.Config {
				return attachments.Config{
					MaxSize: cfg.Attachments.MaxSize,
				}
			},
			func(cfg Config) webhooks.Config {
				return webhooks.Config{
					Secret:         cfg.Webhooks.Secret,
					BotUserEmail:   cfg.Webhooks.BotUserEmail,
					ActionKeywords: cfg.Webhooks.ActionKeywords,
				}
			},
			func(cfg Config) webauthn.Config {
				return webauthn.Config{
					RPDisplayName: cfg.WebAuthn.RPDisplayName,
					RPID:          cfg.WebAuthn.RPID,
					RPOrigins:     cfg.WebAuthn.RPOrigins,
				}
			},
			func(cfg Config) oauth.Config {
				return oauth.Config{
					ClientID:           cfg.OAuth.ClientID,
					ClientSecret:       cfg.OAuth.ClientSecret,
					TokenEncryptionKey: cfg.OAuth.TokenEncryptionKey,
				}
			},
		),
	)
}
