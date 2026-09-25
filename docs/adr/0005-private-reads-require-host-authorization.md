# Require current host authorization for private reads

Compliant peers must obtain fresh authorization from a community's host before serving private or invitation-only content, even when they already hold a cached copy. We chose this over offline private reading because the host owns current membership and sharing permissions; host outages therefore make private reads unavailable, while public cached content can remain readable. This cannot recall previously downloaded content, responses already in flight, or copies retained by remote operators.
