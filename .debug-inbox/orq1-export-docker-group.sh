#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  omniroute)
    images=(
      diegosouzapw/omniroute:latest
      omniroute:3.8.48-affinity-fix
      omniroute:3.8.48-affinity-fix2
      sha256:ebcb21ebf13252d221f14be944456fd7b19081e104f6935a1a3a2f036a7dd91e
    )
    ;;
  web)
    images=(
      ghcr.io/multica-ai/multica-web:latest
      multica-web:20260724-web-cf8017e3d2fd
      multica-web:orq17-stage1-rollback-cf8017e3d2fd
      multica-web:rollback-pre-orq18-20260728
      multica-web:reasoning-ui-20260727T013129Z
      multica-web:gtm-orq18-9f7b963-20260728
      multica-web:transition-6a2aba3
      sha256:1a81143fba5c8ff8b8ca78e2cbce9d9cb0c41a0bc2e29658fafbe8b725c80a1d
      sha256:5e8882da1a85dd6bcf8cbe7dac1861f42bb26eb3a1cc0ee6462126bf06d9813e
      sha256:f0fa70a2e80479b4dffd8d212deab08a2d00e03b97ee6c118f7ab9d4abae515d
    )
    ;;
  backend)
    images=(
      alpine:latest
      pgvector/pgvector:pg17
      multica-backend:transition-6a2aba3
      multica-backend:gtm-reasoning-54ea5e8-20260728
      multica-backend:gtm-cost-e6e0418-20260728
      multica-backend:agy-status-20260727T102815Z
      multica-backend:orq17-stage2-rollback-60133934d8f9
      multica-backend:orq26-default-squad-8227241-20260729
      multica-backend:rollback-before-orq58-8227241-20260729
      multica-backend:rollback-before-orq58-922b138-20260729
      multica-backend:orq134154-63ead4d-20260729
      multica-backend:20260724-srv-a05415c4-daemon-b635556
      multica-backend:t20obs-8f524054
      multica-backend:orq58-canonical-112e8da-20260729
      multica-backend:rollback-before-orq68-112e8da-20260729
      multica-backend:orq68-reconcile-1562638-20260729
      multica-backend:orq134154-edd7b93-20260730
      sha256:04217fa1fdbe1ef9380db76ca086c63ab1d60c9e3396103339e651f6734931e2
      sha256:19142eeeb6a5c6d3155a77c887a015b60996226f663060332682105c509e9973
      sha256:4b72e359eab34196e0565d3b16bda7ac1bd583d6c7d3abdcb58d8d72544fad91
      sha256:60133934d8f99c72e05abc171304e5a7a41ce6827ef4590a873f5642f38bf101
      sha256:922b13862036d906a1ad5cde3e1615adad45393edb13896d2b146753b8384ab6
      sha256:958e82af610c97d3d4fd5e496bf1f1a6ef696e164b0c7cc35c0e95468800c69e
      sha256:994aa2284b55de6d0252899468f235543b721d07d9ac5c89455c11edd26087ec
      sha256:e14f5c35d0640ec4c955efd8d5bbbb6cf219a4d544faafc1fd0f028c5bafcf13
      sha256:ee1c4b714c78079e50de840201c892acd08a037f1b916d02e2fefe60eac1f27c
      sha256:f9e6b777209caf87026a23332a6dfdba0f115a0cbe42f594099ad6c1cb0404c0
    )
    ;;
  *)
    printf 'usage: %s {omniroute|web|backend}\n' "$0" >&2
    exit 2
    ;;
esac

exec docker image save "${images[@]}"
