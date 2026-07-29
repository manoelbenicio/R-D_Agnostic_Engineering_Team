# Owner ruling — centralized Git push authority

- **Effective:** 2026-07-28T20:50Z
- **Authority:** Owner + General Tech Manager

## Ruling

Agents may create local commits only in their owned branch/worktree. They must not
push, force-push, create remote branches, open pull requests, merge or rewrite a
remote ref.

The General Tech Manager is the sole remote Git publisher and will:

1. verify exact local branch and commit SHA;
2. check worktree ownership and dirty state;
3. scan the outgoing range for secrets and oversized blobs;
4. publish one ref at a time without force;
5. verify the remote SHA after each push;
6. record the checkpoint mapping and return the lane to normal local work.

Remote checkpoint publication does not imply review PASS, integration approval,
merge readiness or production deployment. WIP and blocked branches remain visibly
separate from reviewed branches.

This ruling prevents concurrent pushes, accidental upstream changes and remote-ref
races while preserving every material local commit against ORQ2 disk loss.
