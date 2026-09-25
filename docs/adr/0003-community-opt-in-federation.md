# Require server approval and community opt-in for federation

Both server operators must approve a peer connection, and each community must independently opt in before sharing data with that peer. Community sharing is directional: it does not require reciprocal sharing. Server-wide peering alone is insufficient, preserving community control at the cost of extra authorization and revocation work.

Revoking sharing stops new deliveries and subsequent community actions from the peer; contributions and membership records remain, but affected access is suspended and requires explicit reactivation after sharing resumes. Remote deletion cannot be guaranteed. Public webpages remain anonymously readable: these restrictions control federation and remote participation, not copying publicly accessible content.

Amendment (2026-09-25): after revocation, a narrow authenticated cleanup channel still accepts deletion requests, deletion markers, and revocation notices from a trusted peer identity. It cannot return private content, create or edit contributions, grant permissions, or reactivate memberships, and it may be refused when the peer's keys are no longer trustworthy. Without it, account closure and contribution deletion would conflict with revocation. Remote erasure remains unguaranteed.
