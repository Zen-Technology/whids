# CI/CD workflows

| Workflow | Trigger | What it does |
| --- | --- | --- |
| `ci.yml` | push to `master`/`main`/`zen/**`, every PR, manual | gofmt, `go mod tidy` check, `go vet` (linux + windows), staticcheck, race-enabled Linux tests with coverage, full Windows test suite, cross-compiled binaries, govulncheck for windows + linux |
| `codeql.yml` | push to default branch, PRs, weekly, manual | CodeQL `security-extended` for Go (on Windows so the agent code is analysed) and for the workflow files |
| `integration-windows.yml` | manual, weekly, PRs touching `agent/`, `sysmon/`, `utilities/whids/` | Downloads Sysmon (Authenticode-verified) and osquery, then runs `TestSysmonInfo`, `TestAgent` and `TestReporting` on a real Windows host |
| `release.yml` | tag `v*`, manual dry run | Tests, builds the release bundle (agent x86/x64, `manage.bat`, Sysmon configs, checksum-verified Gene rules, `whids-man` for 5 targets), `SHA256SUMS`, build-provenance attestation, then publishes a GitHub release for tags |
| `dependabot.yml` | weekly | Grouped Go module and GitHub Actions updates |

## Enabling Actions on this fork

GitHub disables workflows on forks until someone turns them on:

1. Open **Actions** on the repository and click **I understand my workflows, go ahead and enable them**.
2. For CodeQL results: **Settings > Code security > Code scanning** is enabled automatically on the first upload for public repositories.
3. For releases: no secrets are needed, `release.yml` uses the built-in `GITHUB_TOKEN`.

## Cutting a release

```sh
git tag v2.1.0
git push origin v2.1.0
```

To check a build without publishing: **Actions > Release > Run workflow** with a version such as `2.1.0-rc.1`. The bundle appears as the `release-bundle` artifact.

Verify a downloaded binary:

```sh
sha256sum -c SHA256SUMS --ignore-missing
gh attestation verify whids-v2.1.0-amd64.exe --repo Zen-Technology/whids
```

## Running the same checks locally

```sh
# portable packages (the agent itself only builds for Windows)
PKGS="./agent/config ./agent/sysinfo ./api/... ./event ./ioc ./logger ./los ./sysmon ./tools ./utils/... ./utilities/manager"
gofmt -l .
go vet $PKGS && GOOS=windows go vet ./...
go test -race -count=1 $PKGS
GOOS=windows go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

Regenerate the OpenAPI definition served by the manager (rewrites `api/server/openapi_def.go`):

```sh
WHIDS_UPDATE_OPENAPI=1 go test ./api/server -run 'TestOpenApi'
```

## Test fixtures

* `testdata/events.json`: EDR events shared by the `event`, `logger` and `api/server` suites (see `testdata/README.md`).
* `api/server/main_test.go` generates a throw-away TLS certificate and pins its fingerprint, so no key material is committed.
* `agent/data/windows.osqueryi.exe` and `sysmon/data/Sysmon64.exe` are not committed. Tests that need them skip when they are missing, and the integration workflow downloads them.
