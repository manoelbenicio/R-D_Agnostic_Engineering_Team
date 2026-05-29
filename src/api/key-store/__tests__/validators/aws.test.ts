import { describe, it, expect } from 'vitest';
import { http, HttpResponse } from 'msw';
import { server } from '@/api/__tests__/msw/server';
import { validateAWS } from '../../validators/aws';

describe('validateAWS', () => {
  it('correctly constructs and signs a SigV4 request to AWS STS', async () => {
    let capturedHeaders: Headers | null = null;
    let capturedBody: string | null = null;

    server.use(
      http.post('https://sts.amazonaws.com/', async ({ request }) => {
        capturedHeaders = request.headers;
        capturedBody = await request.text();
        return HttpResponse.xml(
          `<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
            <GetCallerIdentityResult>
              <Arn>arn:aws:iam::123456789012:user/test-user</Arn>
              <UserId>AIDAJQABLZS4A3QDU576Q</UserId>
              <Account>123456789012</Account>
            </GetCallerIdentityResult>
          </GetCallerIdentityResponse>`
        );
      })
    );

    const res = await validateAWS('mock-access-key-id', 'mock-secret-access-key');
    expect(res.ok).toBe(true);
    expect(res.models).toEqual(['q-developer', 'kiro-agent-v1']);

    expect(capturedBody).toBe('Action=GetCallerIdentity&Version=2011-06-15');
    expect(capturedHeaders).not.toBeNull();
    if (capturedHeaders) {
      const authHeader = (capturedHeaders as Headers).get('authorization') || '';
      expect(authHeader).toContain('AWS4-HMAC-SHA256');
      expect(authHeader).toContain('Credential=mock-access-key-id');
      expect(authHeader).toContain('SignedHeaders=content-type;host;x-amz-date');
      expect(authHeader).toContain('Signature=');
    }
  });

  it('handles AWS validation failures gracefully', async () => {
    server.use(
      http.post('https://sts.amazonaws.com/', () => {
        return new HttpResponse(
          `<ErrorResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
            <Error>
              <Type>Sender</Type>
              <Code>InvalidClientTokenId</Code>
              <Message>The security token included in the request is invalid.</Message>
            </Error>
          </ErrorResponse>`,
          { status: 403, headers: { 'content-type': 'text/xml' } }
        );
      })
    );

    const res = await validateAWS('invalid-access-key-id', 'invalid-secret-access-key');
    expect(res.ok).toBe(false);
    expect(res.error).toContain('AWS validation failed (403)');
    expect(res.error).not.toContain('invalid-secret-access-key'); // Must not leak the secret!
  });
});
