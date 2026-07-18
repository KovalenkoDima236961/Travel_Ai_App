# Changelog

All notable changes are recorded here following [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This project uses one release version for the whole monorepo.

## [Unreleased]

### Added

- Release management and versioning v1: release metadata, version endpoints, tagged image builds, staging-like Compose verification, and release playbooks.

### Changed

- None.

### Deprecated

- None.

### Removed

- None.

### Fixed

- None.

### Security

- None.

### Migration Notes

- None.

### API Contract Changes

- Added public, non-sensitive `GET /version` metadata endpoints for all API services and the Web App.

### Known Issues

- Production deployment remains a deliberate manual operation; the repository prepares and verifies artifacts but does not deploy them automatically.

## [0.1.0] - 2026-07-18

### Added

- Initial project release baseline.

### Changed

- None.

### Deprecated

- None.

### Removed

- None.

### Fixed

- None.

### Security

- None.

### Migration Notes

- None.

### API Contract Changes

- Initial OpenAPI contracts and generated Web App client types.

### Known Issues

- None.
