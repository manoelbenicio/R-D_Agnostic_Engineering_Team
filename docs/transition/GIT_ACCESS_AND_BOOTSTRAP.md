# Git Repository Access and New-Environment Bootstrap

## Repository URLs

HTTPS:

```text
https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git
```

SSH:

```text
git@github.com:manoelbenicio/R-D_Agnostic_Engineering_Team.git
```

Browser:

```text
https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team
```

Anonymous HTTPS read access was verified on 2026-07-20. Cloning/fetching does not require a credential. Push access requires a GitHub account with write permission.

## Canonical refs

| Purpose | Ref |
|---|---|
| Default stable branch | `main` |
| Current transition/development branch | `integration/dev-transition-candidate-20260719` |
| Exact deployed source | tag `dev-deploy-20260719-candidate` |
| Original transition handoff | `transition/dev-handoff-20260719` |
| Recovery points | `dev-freeze-20260719-*` tags and `backup/dev-transition-*` branches |

At the start of this guide, `main` resolved to `b6571299b00c8e388abefe7ef9dcbcf8ac715d7f`. Always fetch and read the current remote candidate tip rather than assuming the documentation’s commit is still latest.

## Read-only clone

```bash
cd /home/dataops-lab
git clone https://github.com/manoelbenicio/R-D_Agnostic_Engineering_Team.git
cd R-D_Agnostic_Engineering_Team
git fetch --all --tags --prune
git switch --create integration/dev-transition-candidate-20260719 \
  --track origin/integration/dev-transition-candidate-20260719
git status --short --branch
```

## Configure commit identity

Repository-local configuration used during transition:

```bash
git config user.name mbenicios
git config user.email mbenicios@users.noreply.github.com
```

Use the new operator’s approved GitHub identity if different. Do not reuse another person’s author identity.

## Configure push access

### Option A: GitHub CLI

```bash
gh auth login --hostname github.com --git-protocol https
gh auth status
```

Authenticate interactively as the authorized GitHub user. Do not paste tokens into agent chat or repository files.

### Option B: SSH

Generate a dedicated key on the new environment:

```bash
install -d -m 700 "$HOME/.ssh"
ssh-keygen -t ed25519 -C '<authorized-email>' -f "$HOME/.ssh/id_ed25519_github"
```

Add the public key to the authorized GitHub account, configure SSH, then test:

```bash
ssh -T git@github.com
git remote set-url origin git@github.com:manoelbenicio/R-D_Agnostic_Engineering_Team.git
```

Never transfer the private key through Git or chat.

### Option C: Git Credential Manager

The current WSL host successfully pushed using Windows Git Credential Manager:

```bash
git -c 'credential.helper=!"/mnt/c/Program Files/Git/mingw64/bin/git-credential-manager.exe"' \
  push origin <branch>
```

Use this only when the new environment also has that Windows path and the owner’s authenticated credential-manager session. Native Linux environments should prefer GitHub CLI, SSH, or Linux Git Credential Manager.

## Verify write permission without changing protected branches

Do not test by pushing to `main`. Create a temporary uniquely named branch from the candidate:

```bash
git switch integration/dev-transition-candidate-20260719
git pull --ff-only
git switch --create access-check/<operator>-$(date +%Y%m%d)
git push --set-upstream origin HEAD
```

After write access is confirmed, request deletion of the temporary remote branch according to repository governance. Do not delete recovery branches/tags.

## Verify immutable source anchors

```bash
git fetch --all --tags --prune
git cat-file -e dev-deploy-20260719-candidate^{}
git rev-parse dev-deploy-20260719-candidate^{}
git tag -l 'dev-freeze-*' 'dev-deploy-*' | sort
git branch -r | sort
```

Expected deployed source:

```text
6a2aba3550aaf6b0468a37bfdf2f00c7faaae084
```

## Protect the transition history

- Do not force-push the candidate branch.
- Do not delete/rewrite `dev-freeze-*` or `dev-deploy-*` tags.
- Do not delete/rewrite `backup/dev-transition-*` branches.
- Do not merge candidate into `main` until required gates close.
- Use one branch/worktree per active task lane.
- Require commit SHA, tests, evidence path, and push confirmation for every agent handoff.

## First commands after clone

```bash
cat docs/transition/README.md
cat docs/transition/DEV_RESTART_DOSSIER_20260719.md
cat docs/transition/DATABASE_BACKUP_AND_RESTORE_20260720.md
cat docs/transition/DOCKER_AND_REDIS_INVENTORY_20260719.md
cat docs/transition/SECRETS_AND_ACCESS_REGISTER_20260719.md
cat docs/transition/FRESH_ENV_RESTART_RUNBOOK.md
openspec list
git status --short --branch
```

Do not start agents until source, database backups, secrets, Docker ownership, OpenSpec counts, and active task ownership have been reconciled.

## Access troubleshooting

### Clone works but push fails

The repository is readable publicly, but the authenticated account lacks write permission or Git is not using the intended credential helper. Confirm:

```bash
git remote -v
gh auth status
ssh -T git@github.com
```

Only run the command matching the chosen authentication method.

### Repository access denied

The GitHub repository owner must grant the GitHub user/team write access. An agent cannot create or share that authorization.

### Wrong credentials cached

Do not delete or replace credentials without owner authorization. Inspect which helper is configured:

```bash
git config --show-origin --get-all credential.helper
```

The owner then decides whether to switch account, clear the helper, or use SSH.

## Handoff information to share with the new environment operator

- Repository HTTPS/SSH URL.
- Authorized GitHub username/team.
- Chosen authentication method.
- Candidate branch name.
- Deployed immutable tag.
- Transition documentation index.
- Encrypted database backup package and its separate transfer instructions.
- Secret transfer/regeneration plan.
- Prohibition on pushing directly to `main` or rewriting recovery refs.
