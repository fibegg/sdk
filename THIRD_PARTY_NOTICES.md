# Third-party notices

This project depends on third-party Go modules listed by `go list -m all` and
recorded in `go.mod`/`go.sum`. Their licenses remain their own. The direct
runtime and build dependencies in the current release are:

| Module | Version | License |
| --- | --- | --- |
| `github.com/BurntSushi/toml` | v1.6.0 | MIT |
| `github.com/PaesslerAG/jsonpath` | v0.1.1 | BSD-3-Clause |
| `github.com/coder/websocket` | v1.8.15 | ISC |
| `github.com/google/uuid` | v1.6.0 | BSD-3-Clause |
| `github.com/magefile/mage` | v1.17.2 | Apache-2.0 |
| `github.com/mark3labs/mcp-go` | v0.56.0 | MIT |
| `github.com/santhosh-tekuri/jsonschema/v6` | v6.0.2 | Apache-2.0 |
| `github.com/spf13/cobra` | v1.10.2 | Apache-2.0 |
| `github.com/spf13/pflag` | v1.0.10 | BSD-3-Clause |
| `github.com/stretchr/testify` | v1.11.1 | MIT |
| `gopkg.in/yaml.v3` | v3.0.1 | MIT and Apache-2.0 |

Release SPDX and CycloneDX SBOMs provide the exact transitive dependency
inventory for each artifact. The source distribution retains each module's
license text in the Go module cache or vendored source when vendoring is used.
The project MIT license applies only to original repository content.
