package email

// This file is intentionally small. The email package is constructed via
// NewSender (provider selection) in the composition root, mirroring the
// self-contained "module" layout used by the other Notification Service
// packages. Keeping a module.go here keeps the package shape familiar across the
// codebase even though no DI framework is used.
