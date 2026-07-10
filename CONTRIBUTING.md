# Contributing

Thank you for helping improve WoW Markets. Open an issue before a large or
contract-breaking change so client and service compatibility can be planned.

## Development

Fork the repository, create a focused branch, and run `make check` before
opening a pull request. Do not use production installation tokens, upload test
or synthetic scans, or include real SavedVariables in fixtures. Tests must use
invented identities and local HTTP servers.

Contract changes require updated specifications and fixtures and must preserve
the staged compatibility order documented by the project maintainers.

By submitting a contribution, you agree that it is licensed under Apache-2.0.
Be respectful and follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
