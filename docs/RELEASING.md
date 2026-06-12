# Releasing the Code Sweeper CLI

The `membrane` CLI is released with [GoReleaser](https://goreleaser.com) from
`clients/cli/.goreleaser.yaml`. One tag produces the full packaging matrix:

| Channel | Artifact |
| --- | --- |
| Tarball / zip | `membrane_<ver>_<os>_<arch>.{tar.gz,zip}` for linux/darwin/windows × amd64/arm64 |
| Debian | `membrane_<ver>_<arch>.deb` (nfpm) |
| RPM | `membrane-<ver>.<arch>.rpm` (nfpm) |
| Homebrew | formula pushed to `Ozgurisikdamar/homebrew-tap` |
| winget | manifest PR to `microsoft/winget-pkgs` |
| Checksums | `checksums.txt` |

The CLI binary stamps `main.version` via ldflags, so `membrane version` prints
the released tag.

## Cut a release

```sh
git tag v0.1.0
git push origin v0.1.0   # the CI `release` job runs GoReleaser on tags matching v*
```

Or locally (needs `goreleaser` + a `GITHUB_TOKEN`): `task release`.
Validate the config without releasing: `task release:check`.
Build artifacts locally without publishing: `task release:snapshot` → `clients/cli/dist/`.

## One-time prerequisites

- **Homebrew tap**: create the repo `Ozgurisikdamar/homebrew-tap`; the release
  token needs write access to it.
- **winget**: GoReleaser opens a PR against `microsoft/winget-pkgs` from a fork
  the token can push to.
- **CI**: the `release` job in `deploy/ci/github-ci.yml` only runs on `v*` tags
  and uses the built-in `GITHUB_TOKEN` (mirror to `.github/workflows/` per D-026).
