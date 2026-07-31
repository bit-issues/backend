// Golang project template
//
//	@title						Project API
//	@version					1.0.0
//	@description				Project API documentation
//
//	@contact.name				API Support
//	@contact.url				https://github.com/capcom6
//	@contact.email				i@capcom.me
//
//	@license.name				Apache 2.0
//	@license.url				http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@host						localhost:3000
//	@BasePath					/api/v1
//
//	@securitydefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
package main

import (
	"runtime"
	"strconv"

	"github.com/bit-issues/backend/internal"
	"github.com/go-core-fx/healthfx"
	"github.com/samber/lo"
)

//go:generate swag init --parseDependency --outputTypes go -g ./main.go -o ./internal/server/docs
//go:generate npm --prefix frontend ci
//go:generate npm --prefix frontend run build

//nolint:gochecknoglobals // build metadata
var (
	appVersion   = "dev"
	appReleaseID = "0"
	appBuildDate = "unknown"
	appGitCommit = "unknown"
	appGoVersion = runtime.Version()
)

func main() {
	internal.Run(healthfx.Version{
		Version:   appVersion,
		ReleaseID: lo.Must1(strconv.Atoi(appReleaseID)),
		BuildDate: appBuildDate,
		GitCommit: appGitCommit,
		GoVersion: appGoVersion,
	})
}
