package users

// This file is intentionally small. The user-lookup client is constructed via
// New in the composition root, mirroring the self-contained "module" layout used
// by the other Notification Service packages (and Trip Service's outbound
// clients). No DI framework is used.
