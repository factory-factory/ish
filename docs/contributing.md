# Contributing

Contributions to **ish** are welcome and appreciated! Contributions could be
code, tests, documentation, or even being a friendly and helpful voice in chat
channels. Please read this project contribution guide to familiarize yourself
with the project structure and development workflow.

In addition to building the best possible issue tracker, it is also vital that
we create and maintain an inclusive, welcoming, and supportive community. In
short, be courteous, be inquisitive, and don't be a jerk. You can read more in
the [community guidelines][cg].

## Getting Started

- Clone the git repo: `git clone git@github.com:factory-factory/ish.git`
- Make sure go version 1.27 or greater is installed.
- Build the project: `go build .`
- Run **ish** to see what work is in the queue: `./ish`

## Dependencies

Third-party dependencies are tracked in `go.mod`. We try to keep external modules to a minimum.

## Testing

Tests follow standard idiomatic go conventions. Test files end with the
`_test.go` suffix and typically mirror the source file name.

Run `go test ./...` on the command line to run all tests in the current
directory and all subdirectories.

## What to Work On

A good place to see what needs work is to install and run **ish** and see what
issues are open. In the future, there will be a roadmap page in the wiki that
discusses new features and future directions for the language.

Questions, comments, and suggestions can be addressed to
[brendan@berg.industries][mailto].

[cg]: community_guidelines.md
[mailto]: mailto:brendan@berg.industries
