#!/usr/bin/env bash
set -euo pipefail

repo_dir="/home/ec2-user/workspace/R-D_Agnostic_Engineering_Team"
stack_name="orq17-bootstrap-owner-secret"
template_file="$repo_dir/.deploy-control/p0/evidence/orq17-bootstrap-owner-secret-stack.yaml"
owner_email="dataops.cloud.mbf@gmail.com"
resolver_role="arn:aws:iam::809809509961:role/cw-agent-orquestradores"

aws cloudformation deploy \
  --profile owner-p0 \
  --region sa-east-1 \
  --stack-name "$stack_name" \
  --template-file "$template_file" \
  --parameter-overrides \
    "OwnerEmail=$owner_email" \
    "ResolverPrincipalArn=$resolver_role" \
  --tags \
    "Environment=production" \
    "Application=multica" \
    "Control=ORQ-17"

aws cloudformation describe-stacks \
  --profile owner-p0 \
  --region sa-east-1 \
  --stack-name "$stack_name" \
  --query "Stacks[0].Outputs[?OutputKey=='SecretArn'].OutputValue" \
  --output text
