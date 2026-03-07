# llint — Lunar Linux Module Linter

A lint checker for Lunar Linux moonbase module files.

## Usage

```
llint [flags] <module-name>
llint --path <module-dir>
```

### Flags

- `--fix` — Auto-fix fixable issues (rewrites files in-place)
- `--verbose` — Show what was fixed (use with `--fix`)
- `--max-line-length N` — Maximum line length for heredoc text in DETAILS (default: 120)
- `--path <dir>` — Lint a module directory directly (skips config and index lookup; useful in CI/GitHub Actions)

### Exit Codes

- `0` — No errors found
- `1` — Lint errors found
- `2` — Usage error (bad arguments, missing config, module not found)

## Module Resolution

`llint` resolves module names the same way other lunar tools do:

1. Check all `zlocal*` sections first (zlocal overrides take priority)
2. Fall back to `module.index` lookup

Configuration is loaded from `/etc/lunar/config`, with `/etc/lunar/local/config` overlaid on top.

## Checks

### DETAILS

| Check                    | Fixable | Description                                                                              |
|--------------------------|---------|------------------------------------------------------------------------------------------|
| `=` alignment            | Yes     | All variable assignments must have `=` at the same column                                |
| Special option placement | Yes     | `PSAFE`, `GARBAGE`, etc. must be flush-left after the main variable block                |
| Heredoc spacing          | Yes     | Exactly one blank line before `cat << EOF` and no extra blank lines after `EOF`          |
| Heredoc line length      | Yes     | Lines in `cat << EOF` block must not exceed `--max-line-length`                          |
| Duplicate assignments    | Yes/No  | Exact duplicates are auto-removed; conflicting duplicates (different values) are errors   |
| Required fields          | No      | `MODULE`, `VERSION`, `SOURCE`, `WEB_SITE`, `ENTERED`, `UPDATED`, `SHORT` must be present |

**Special options** (must be flush-left, after main block, before heredoc):
`PSAFE`, `GARBAGE`, `ARCHIVE`, `KEEP_SOURCE`, `USE_WRAPPERS`,
`COMPRESS_MANPAGES`, `KEEP_OBSOLETE_LIBS`, `LUNAR_RESTART_SERVICES`, `LDD_CHECK`, `FUZZY`

### DEPENDS

| Check                  | Fixable | Description                                                                                             |
|------------------------|---------|---------------------------------------------------------------------------------------------------------|
| Allowed functions only | No      | Only `depends`, `optional_depends`, `optional_depends_requires`, `optional_depends_one_of`              |
| No bash logic          | No      | `if`/`case`/`for`/`while`, test expressions, variable assignments, command substitutions are all errors |

## Output Format

```
module/DETAILS:3: error: '=' not aligned (expected column 16, found 10)
module/DEPENDS:14: error: disallowed bash logic: 'if'
```

## Build

```
cd tools/llint
go build -o llint .
```
