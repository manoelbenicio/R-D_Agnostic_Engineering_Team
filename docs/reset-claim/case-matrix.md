# Reset-Claim Case Matrix

1. **No credit remaining**
   - **Expected behavior**: Try to redeem if within natural reset window logic.
   - **Command**: `prodex redeem <profile>`
   - **Expected outcome**: Success (credit restored) or Failure (quota fully exhausted).

2. **With credit remaining**
   - **Expected behavior**: Do not redeem unnecessarily unless explicitly forced or natural grace period allows.
   - **Command**: `prodex redeem <profile>`
   - **Expected outcome**: Skipped or prompt confirmation required.

3. **Near reset window**
   - **Expected behavior**: Prompt if within 1 hour of weekly reset, unless forced with `--yes`.
   - **Command**: `prodex redeem <profile>`
   - **Expected outcome**: Redeem executed if confirmed or auto-approved.

4. **Weekly quota exhausted**
   - **Expected behavior**: Auto-redeem should check pool eligibility, abort if weekly limit hard-capped.
   - **Command**: `prodex redeem <profile> --auto-redeem`
   - **Expected outcome**: Failure due to weekly exhaustion, no credit consumed.

5. **5-hour window only**
   - **Expected behavior**: Check if the credit strictly resets the 5h window.
   - **Command**: `prodex redeem <profile>`
   - **Expected outcome**: 5-hour quota reset correctly.

6. **All accounts exhausted**
   - **Expected behavior**: Fallback mechanism triggers gracefully.
   - **Command**: `prodex redeem <profile>` across pool
   - **Expected outcome**: All attempts fail cleanly without aggressive retries.

7. **Non-OpenAI provider**
   - **Expected behavior**: Reset-claim is OpenAI/Codex specific.
   - **Command**: `prodex redeem <non_openai_profile>`
   - **Expected outcome**: Command aborted or skipped as unsupported provider.
