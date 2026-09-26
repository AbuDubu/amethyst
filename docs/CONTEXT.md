# Amethyst

Amethyst connects communities around shared interests across independently operated servers.

## Language

**Server**:
An independently operated home for accounts and communities that can communicate with other servers in the network.
_Avoid_: Community, group

**Account**:
A person's identity on one home server, through which they can participate in communities on that server or other servers.
_Avoid_: Person (one person and one account are not necessarily the same thing)

**Local account**:
An account whose home server is this server; only local accounts have credentials, an email address, or a server role here.
_Avoid_: User (ambiguous between a person and an account)

**Remote account**:
An account known to this server whose home server is another server.
_Avoid_: Foreign user, guest

**Home server**:
The server to which an account belongs.
_Avoid_: Community server when referring to an account's home

**Community**:
A shared-interest space hosted on one server, with its own membership and moderation rules, that can include accounts from other servers.
_Avoid_: Server, node

**Community host**:
The server with authority over a community's discussions and membership state.
_Avoid_: Home server when referring to the community rather than an account

**Peer**:
Another server with which operators on both sides have approved a federation connection; this approval alone does not authorize sharing any particular community's data.
_Avoid_: Member (membership belongs to accounts within communities)

**Community sharing approval**:
A community's explicit, directional permission to share its data with a particular approved peer, subject to that community's access rules; no reciprocal community sharing is implied.
_Avoid_: Server peering (which is a separate approval)

**Revocation cleanup**:
Authenticated deletion requests, deletion markers, and revocation notices that may still pass between servers after community sharing is revoked; it never carries new content or restores access.
_Avoid_: Resync, reconnection

**Membership**:
An account's admitted participation in a community, granting the ability to contribute subject to that community's rules.
_Avoid_: Follow, subscription

**Public community**:
A discoverable community whose public webpages anyone may read, with membership available through open enrollment or approval; public web access does not itself grant remote participation or federation delivery.
_Avoid_: Open membership (which describes admission, not visibility)

**Private community**:
A community whose name and description are discoverable but whose content is restricted to members admitted through approval.
_Avoid_: Invitation-only community

**Invitation-only community**:
An unlisted community whose content is restricted to members admitted by invitation.
_Avoid_: Private community (which is discoverable)

**Federation**:
Cooperation between independently operated servers that allows accounts to participate in communities beyond their home server.
_Avoid_: Synchronization (which does not express participation or ownership)

## Participation and discussions

**Community owner**:
The account responsible for a community's settings, moderator appointments, and sharing approvals.
_Avoid_: Server operator (a separate role)

**Community moderator**:
An account appointed to manage community membership requests, reports, content removals, and bans.
_Avoid_: Owner when referring only to moderation authority

**Server operator**:
The person responsible for a server's admission policy, peer connections, and ability to suspend communities hosted there.
_Avoid_: Community moderator

**Discussion**:
A titled community conversation started by an authored contribution, optionally including images, with associated replies.
_Avoid_: Personal status, feed

**Reply**:
An authored contribution within a discussion, optionally responding to another reply in that same discussion.
_Avoid_: Discussion when referring to one response

**Server invitation**:
Permission to register an account on a server; it does not confer community membership.
_Avoid_: Community invitation

**Operator invitation**:
A server invitation that also makes the registering account a server operator; the first one is issued from the command line to bootstrap a new server.
_Avoid_: Admin account

**Community invitation**:
An invitation for an account to become a member of a particular community.
_Avoid_: Server invitation

**Report evidence**:
A restricted snapshot of reported content available to authorized moderators and operators for no more than 30 days after capture, even if the original contribution is deleted.
_Avoid_: Revision history, permanent archive
