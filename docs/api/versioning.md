# API versioning

The API is at contract version `v1`; paths are not URL-versioned. Compatible
additions are preferred: optional response fields, optional request fields and
new endpoints. A field removal/rename, semantic change, required input,
authorization tightening or incompatible pagination change is breaking.

A breaking change requires a contract-changelog entry, API/consumer migration,
updated generated types and fixtures, tests, and release notes. Keep a
deprecated field until all supported Web App releases are migrated; never
silently reuse a removed field name. Review the OpenAPI diff in pull requests
until automated baseline diffing is adopted.
