// Package e2ewiring hosts cross-seam production-wiring integration tests for the
// end-to-end observability contract (OpenSpec 6.2, AB-REQ-39/40). It contains no
// production logic; it exists solely so an external *_test package can import
// every real hop emitter (middleware ingress, service queue/persist, brain
// admission, daemon CLI, gateway route, daemonws delivery) and prove they emit
// real, spec-valid hop spans that assemble into one continuous trace. Keeping
// this a leaf package (imported by nothing) guarantees zero import cycles.
package e2ewiring
