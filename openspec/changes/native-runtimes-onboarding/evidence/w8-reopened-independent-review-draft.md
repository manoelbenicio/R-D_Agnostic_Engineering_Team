# W8 independent-review draft — native onboarding 1.5/1.6/1.7

Status date: 2026-07-19. Reviewer: W8 / Codex (`/root`), evidence-only and
read-only on product code. Because 1.5 was produced by Codex56#A and the original
1.7 producer/reviewer identities are incomplete, this same-family review cannot
repair provenance or act as adjudication. Kiro TL remains adjudicator.

## Classification

| Task | Implemented | Reviewed | Verified | Accepted |
|---|---|---|---|---|
| 1.5 frontend onboarding | yes, candidate slice present | prior independent review + this evidence review | bounded 43-assertion report; callback DOM test, build, E2E and UAT remain non-claims | **NO new acceptance; task remains open** |
| 1.6 design/i18n/web QA | partial | yes | node-only parity/type checks reported; offline production build and jsdom DOM harness blocked | **NO — open** |
| 1.7 backend auth | yes | accepted artifact plus push-eligibility reviews | bounded backend tests previously reproduced | **task is already checked/accepted; atomic push remains HOLD** |

Task 1.7 is not truthfully “reopened” in the current OpenSpec state. Its accepted
backend scope remains checked. The push unit is held because the original producer
and accepting reviewer are unnamed, the executed `auth_routes_test.go` was outside
the accepted manifest, and the shared env-template review remains separate. W8
neither revokes acceptance nor converts push eligibility into acceptance.

## Exact 1.5 source manifest

```text
2af77c72b12d6ac1b39a1dfca61cee6ed7b6c49fca67af53e233daf4293611ef  multica-auth-work/packages/core/auth/service.ts
2add5c81097326164d0e33e51f2d5ad2b7d25bca92d5117af72a71ca52f50e17  multica-auth-work/packages/core/auth/service.test.ts
bd0d7ac9560a04d9e37e0b00d2c659e55f68c06b551cd6ae1872b9140a6d279a  multica-auth-work/packages/core/auth/store.ts
39fdaca276de65bdc8b4fc399069a1f13861f075bea4c0bb62dedf14710f4ee7  multica-auth-work/packages/core/auth/store.test.ts
e10e6945e6d66cf0ef39fa02caf1cddf9bef09b4b6c464200348ffe9b4ca4031  multica-auth-work/packages/core/auth/index.ts
937cadb61759d935dbf226050dffd895de349e67694d9784ff7d2f37a5f755ff  multica-auth-work/packages/views/auth/login-page.tsx
f3b632f9bbd1637c6405ff1869544b792c0c9d11b63455507a3508521e0a536d  multica-auth-work/packages/views/auth/login-page.test.tsx
1c17dba85bb0cba526fb4d1d02d3aa819056a7e7e9321eaf633a48deccabb5df  multica-auth-work/packages/views/auth/auth-locale-parity.test.ts
232a2b9d115cccb7c06d26590429125a6acb919114ef9990e73dee5dd511dbcf  multica-auth-work/packages/views/auth/use-logout.ts
46e1b6a90ae604e0e1360d06ecd2025c8e4c7587a652653648d8e7e21e2eab94  multica-auth-work/packages/views/auth/index.ts
80066e7d47650ebe96bc24ce2172c238de5eeda892fd7d3b99c7f74542c1805c  multica-auth-work/apps/web/test/onboarding-auth-gate.test.ts
a5ead9a772b7a31629190124a4721b62d654f6817a4fce3378f06ac8773bf4c7  multica-auth-work/apps/web/app/(auth)/login/page.tsx
4b176203725a6f2b692b549d54b97a60a9eea3c105030b4d1b515af884aa1d14  multica-auth-work/apps/web/app/(auth)/login/page.test.tsx
14390e9f3c37c4429bb4eaa3f31d250f61960725638131fc25fa9534f81be9fb  multica-auth-work/apps/web/app/page.tsx
c0a0af82e72ba014ef33c6eff1f675b15e9353738cbcc46123967c5338cfaf59  multica-auth-work/packages/core/api/client.ts
ee502aa4323c285967cb40d2d7ef73f3e2b5a0bbf3878fa84e2a34ab96c62fcc  multica-auth-work/packages/core/api/client.test.ts
cd4e36849170df039e88f1e371e40a2402a6fcefd6f35f980c72a0b02ec210e8  multica-auth-work/apps/web/app/auth/callback/page.tsx
83f80167e9d81a81327e6d7b1a529c0cf4a7b0423972c93ea836c4cd4cdda4b1  multica-auth-work/apps/web/app/auth/callback/page.test.tsx
```

## Exact 1.6/1.7 boundary manifest

```text
2997ed83cedbd6ad46fe886e28269ec586ded5e193f698099f59f30683c03dd4  multica-auth-work/apps/web/app/layout.tsx
9535c99f99925d44ac4c5cb05e5ca97b83e64b3cdce12db50ef6c9c47af738cb  multica-auth-work/apps/web/app/globals.css
5377c90de438b509ca698146bff3f6134aa4d3b2f54626134344561870a75d5c  multica-auth-work/packages/ui/styles/tokens.css
7e814662104c09feb6eba8b02d05bf26ddac9e12659bd97a8541d3f2d974446e  multica-auth-work/server/cmd/server/auth_routes_test.go
```

## Exact evidence-input manifest

```text
40b4cb7ac08f3bd3d991964eb606d04ed8648a9a48edc66174a92cc43ee0b86c  .planning/agent-brain-v3/evidence/native-onboarding-1.5-review.md
e59c3a295d0c00219762b37ad889b6dd7736e19ececb126ebf3d030a75bd9311  .deploy-control/evidence/native-onboarding-1.6-acceptance-diagnostic.md
1bc6ca4385ee184b8c7d047732b90ee3ca33a4f8a30dae6ab813e5ed2c818dba  .planning/agent-brain-v3/evidence/native-onboarding-1.7-push-eligibility-independent-review.md
2a5f7368a63202f5decb27bd562589e1cc9ad406499b29a14a471f1c1425c095  .planning/agent-brain-v3/evidence/native-auth-password-provisioning.md
```

Required next evidence: repository-reproducible callback DOM assertions and
web build/UAT for 1.5/1.6 on a Linux-native filesystem with deterministic local
fonts; then distinct review and TL adjudication. For 1.7 push eligibility, name
or explicitly waive the irrecoverable producer/reviewer provenance, extend the
manifest to `auth_routes_test.go`, reconcile the env-template under an authorized
review, and re-hash the atomic unit immediately before integration.

No product, task checkbox, GSD, environment file, secret, auth file, service,
DB, or network state was changed or accessed by this draft.
