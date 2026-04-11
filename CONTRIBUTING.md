# Contributing

Thanks for your interest in burrow. Contributions are welcome — bug fixes,
new features, documentation improvements, and test coverage are all appreciated.

## Getting started

```bash
git clone https://github.com/xenomorphingtv/burrow
cd burrow
make build   # compile
make check   # vet + test
```

Requires Go 1.22+.

## Before opening a pull request

- Run `make check` and make sure everything passes
- Keep commits focused — one logical change per commit
- If you're adding a feature, add tests for it
- If you're fixing a bug, a test that would have caught it is a bonus

## Project structure

```
cmd/burrow/        # CLI entry point and subcommand files
internal/
  config/          # TOML loading and merging
  runner/          # executor, pipeline resolver, scheduler, pool
  store/           # bbolt run history
  tui/             # Bubbletea model, rendering, keybindings
  daemon/          # background daemon, IPC, service install
  notify/          # desktop notifications
  version/         # build-time version string
```

## Reporting bugs

Open an issue on GitHub. Include:

- What you ran
- What you expected
- What actually happened
- Your OS and distro
- Output of `burrow version`

## Feature requests

Open an issue describing what you want and why. Check the existing issues first
to avoid duplicates.

## Code style

Standard Go. Run `go vet ./...` before pushing. No linter configuration is
enforced, but code should be readable and consistent with the surrounding style.
