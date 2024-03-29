schema_version = 1

project {
  license           = "Apache-2.0"
  copyright_holder  = "The Flux authors"
  copyright_year    = 2024

  # (OPTIONAL) A list of globs that should not have copyright/license headers.
  # Supports doublestar glob patterns for more flexibility in defining which
  # files or folders should be ignored
  header_ignore = [
    ".github/**",
    ".goreleaser.yml",
  ]
}
