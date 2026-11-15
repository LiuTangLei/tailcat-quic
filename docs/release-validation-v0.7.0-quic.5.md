# v0.7.0-quic.5 candidate validation

This revision pins public immutable shared QUIC `v0.65.0-quic.4` and the dedicated Tailcat core `v1.102.5-tailcat.2`. The root module explicitly retains the LiuTangLei uTLS fork. The dedicated core is not replaced by generic Tailscale main.

The runtime changes cancel failed proxy copies, immediately abort failed reliable streams while preserving the shared authenticated connection, and reduce proxy helper creation. Cancellable pure-DERP queue backpressure preserves the existing32-packet queue, BBRv3 and pacing. Normal final-byte/FIN delivery and native WireGuard send behavior are retained. Intentional queue closure is not a transport error, and cancellation is advertised only when supported by the host.

Relevant exact-source race tests, ordinary tests, vet and native Linux contract checks passed before packaging. Final clean builds, package identities and their runtime smoke receipts are recorded in the release manifest and validation receipt. A microbenchmark is not a universal CPU/GC or throughput claim.

The owner confirmed that TCP between X and the test hosts is inherently slow. Real DERP WAN retries, throughput and lifecycle results on that path are excluded from this release's gate; original failures remain archived. This is not a claim that those measurements passed. Physical mobile-network and operator low-MTU validation is not claimed.

This document describes a release candidate. It does not publish an application release or approve production upgrades. Old-source JP24h observations remain historical; they are not a soak of this candidate. Standalone macOS packages use an ad-hoc signature, not Developer ID notarization.
