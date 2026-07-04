# Go Integration - Policy Push

Status: PRE-DEPLOY REQUIRED

## 1. Purpose

Go pushes desired runtime envelope. Rust/prodex executes within it.

## 2. Policy Fields

Required:

- tenant id;
- policy id;
- allowed providers;
- allowed profiles;
- default model/provider;
- Smart Context mode;
- auto redeem mode;
- gateway mode;
- budgets;
- kill switches;
- audit settings.

## 3. Prohibited Fields

Policy payload must not include:

- raw OAuth token;
- raw API key;
- database URL;
- `auth.json` contents;
- cookies.

## 4. Apply Rules

- idempotent by policy id;
- rejected on unknown provider capability;
- rejected when shared SQLite configured;
- rejected when kill switch service unavailable;
- event emitted on apply.

