// Package entitlement is the domain layer for tenant feature entitlements.
//
// Scope:
//   - Features registered in platform_features are gated by plan / override / snapshot.
//   - Other product modules are controlled by RBAC and menus.
//
// Resolution order (low -> high):
//
//	plan defaults < addon grants < tenant overrides
//
// Runtime consumers should use Present / Runtime helpers against the same EffectiveSet.
package entitlement
