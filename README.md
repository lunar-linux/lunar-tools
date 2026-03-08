# lunar-tools

A collection of system administration tools for [Lunar Linux](https://lunar-linux.org/).

## Tools

### Shell tools (prog/)

- **clad** -- Cluster administration. Run jobs serialized or parallelized across groups of machines via SSH.
- **lids** -- Lunar intrusion detection. Permission/owner checking and md5sum verification of installed modules.
- **lmodules** -- Kernel module management. Ncurses interface for maintaining the list of needed system kernel modules.
- **lnet** -- Network configuration. Ncurses frontend for managing network devices, gateways, routing, hostname, and DNS.
- **lservices** -- Startup services. Ncurses interface to start, stop, enable, and disable startup services.
- **ltime** -- Timezone management. Change timezone and GMT/localtime settings.
- **luser** -- User administration. Ncurses tool for managing passwd, shadow, and group files.
- **installkernel** -- Kernel installation. Install kernels and modules from a kernel source tree with bootloader updates.
- **run-parts** -- Run all scripts in a directory.

### Go tools (tools/)

- **llint** -- Lunar module linter. Validates moonbase module files (DETAILS, DEPENDS) for correctness. See [tools/llint/README.md](tools/llint/README.md) for details.

## Development

### Prerequisites

- GNU Make
- Go 1.24+ (for building Go tools)
- Git

### Building Go tools

```bash
make build-tools                    # build for host architecture
make build-tools GOARCH=amd64      # cross-compile for amd64
make build-tools GOARCH=386        # cross-compile for i686
```

### Running tests

```bash
cd tools/llint && go test ./...    # run llint tests
cd tools/llint && go vet ./...     # static analysis
```

### Installing

```bash
make install                       # install shell tools only (legacy)
make install-all                   # install shell tools + Go binaries
make install-all DESTDIR=/tmp/pkg  # install to staging directory
```

## Releasing

Releases are built automatically by GitHub Actions when a version tag is pushed. Tags follow the `<year>.<counter>` format (e.g., `2025.1`, `2025.2`).

The workflow runs Go tests, cross-compiles for amd64 and i686, and creates a GitHub release with two architecture-specific tarballs:

- `lunar-tools-<version>-amd64.tar.xz`
- `lunar-tools-<version>-i686.tar.xz`

Each tarball contains the full source tree plus the pre-compiled Go binaries. After extracting, run `make install-all` to install everything.

To build a release tarball locally:

```bash
make release VERSION=2025.1 GOARCH=amd64 ARCH=amd64
make release VERSION=2025.1 GOARCH=386 ARCH=i686
```

## License

See [COPYING](COPYING) for license details.
