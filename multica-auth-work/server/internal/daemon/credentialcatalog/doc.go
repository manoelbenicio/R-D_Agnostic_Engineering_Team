// Package credentialcatalog discovers provider credential homes using only
// filesystem metadata beneath an owner-controlled root.
//
// It does not open credential artifacts, parse account identity, mutate source
// homes, delete files, or run filesystem watchers. Raw paths remain daemon-local.
package credentialcatalog
