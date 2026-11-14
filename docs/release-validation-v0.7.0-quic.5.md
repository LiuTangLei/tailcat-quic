# Tailcat-QUIC v0.7.0-quic.5 validation scope

This revision moves the shared library from QUIC `v0.63.0-quic.3` to
`v0.65.0-quic.2`. The latter includes the reviewed upstream 0.65 merge and a
fix for concurrent public address reads during client path migration.

The application keeps its dedicated Tailscale integration core, published as
`v1.102.5-tailcat.1`. This preserves its peer configuration and immediate peer
revocation APIs; the core adapts the two authenticated CONNECT handlers to the
new HTTPStream error return. Both module roots explicitly pin the compatible
uTLS fork. Release packages must contain only public immutable module pins.

Mandatory connection secrets, node authentication, reliable TCP streams,
BBRv3, pacing, final-byte/FIN acknowledgment and explicit abort remain.
This is not a claim that the Tailscale DATAGRAM throughput results also apply
to Tailcat's reliable-stream workload.

Local checks cover ordinary and race application tests, SSH/SFTP and CLI
contracts, real Chrome transfers in both directions, and the dedicated core's
authentication, stream lifetime and shutdown behavior. Package verification
must additionally exercise actual extracted native executables and verify
all seven archives and six Linux packages against their exact build identity.

The accompanying `build.json` and `validation.json` identify final source,
dependencies, executable hashes, exact completed checks, retained failures,
both-direction WAN samples and validation limits. A build or local test is
not by itself a WAN acceptance result. Publication and installer changes
follow the completed evidence review; existing release tags stay immutable.

No universal throughput or CPU improvement is claimed. Existing BBR ProbeRTT
latency and post-startup burst recovery limitations are outside this revision.
Windows and ARM cross-builds are not device execution, and cellular testing
is outside this server/desktop release scope.
