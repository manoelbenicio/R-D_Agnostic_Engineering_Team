/* eslint-disable agentverse/no-sideways-capability-imports */
import { appFetch } from '@/shell/app-fetch';

function bufToHex(buffer: ArrayBuffer): string {
  return Array.from(new Uint8Array(buffer))
    .map((x) => ('00' + x.toString(16)).slice(-2))
    .join('');
}

async function hmac(key: ArrayBuffer | Uint8Array, message: string | Uint8Array): Promise<ArrayBuffer> {
  const msgBuffer = typeof message === 'string' ? new TextEncoder().encode(message) : message;
  
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    key as ArrayBuffer,
    { name: 'HMAC', hash: { name: 'SHA-256' } },
    false,
    ['sign']
  );
  
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return await crypto.subtle.sign('HMAC', cryptoKey, msgBuffer as any);
}

export async function validateAWS(
  accessKeyId: string,
  secretAccessKey: string
): Promise<{ ok: boolean; error?: string; models?: string[] }> {
  try {
    const service = 'sts';
    const region = 'us-east-1';
    const host = 'sts.amazonaws.com';
    const endpoint = `https://${host}/`;
    
    // Formatting dates
    const now = new Date();
    const amzDate = now.toISOString().replace(/[:-]|\.\d{3}/g, '');
    const dateStamp = amzDate.substring(0, 8);
    
    const requestBody = 'Action=GetCallerIdentity&Version=2011-06-15';
    const bodyEncoded = new TextEncoder().encode(requestBody);
    const bodyHashBuffer = await crypto.subtle.digest('SHA-256', bodyEncoded);
    const bodyHash = bufToHex(bodyHashBuffer);
    
    // Canonical Request details
    const method = 'POST';
    const canonicalUri = '/';
    const canonicalQueryString = '';
    
    const contentType = 'application/x-www-form-urlencoded; charset=utf-8';
    
    const canonicalHeaders = 
      `content-type:${contentType}\n` +
      `host:${host}\n` +
      `x-amz-date:${amzDate}\n`;
      
    const signedHeaders = 'content-type;host;x-amz-date';
    
    const canonicalRequest = 
      `${method}\n` +
      `${canonicalUri}\n` +
      `${canonicalQueryString}\n` +
      `${canonicalHeaders}\n` +
      `${signedHeaders}\n` +
      `${bodyHash}`;
      
    const canonicalRequestHashBuffer = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(canonicalRequest));
    const canonicalRequestHash = bufToHex(canonicalRequestHashBuffer);
    
    // String to Sign
    const algorithm = 'AWS4-HMAC-SHA256';
    const credentialScope = `${dateStamp}/${region}/${service}/aws4_request`;
    const stringToSign = 
      `${algorithm}\n` +
      `${amzDate}\n` +
      `${credentialScope}\n` +
      `${canonicalRequestHash}`;
      
    // Derived signing key
    const secretPrefix = new TextEncoder().encode('AWS4' + secretAccessKey);
    const kDate = await hmac(secretPrefix, dateStamp);
    const kRegion = await hmac(kDate, region);
    const kService = await hmac(kRegion, service);
    const kSigning = await hmac(kService, 'aws4_request');
    
    // Signature
    const signatureBuffer = await hmac(kSigning, stringToSign);
    const signature = bufToHex(signatureBuffer);
    
    const authorization = 
      `${algorithm} ` +
      `Credential=${accessKeyId}/${credentialScope}, ` +
      `SignedHeaders=${signedHeaders}, ` +
      `Signature=${signature}`;
      
    // POST request to STS using appFetch
    const res = await appFetch(endpoint, {
      method: 'POST',
      headers: {
        'content-type': contentType,
        host,
        'x-amz-date': amzDate,
        authorization,
      },
      body: requestBody,
    });
    
    if (!res.ok) {
      const errText = await res.text();
      return { ok: false, error: `AWS validation failed (${res.status}): ${errText}` };
    }
    
    return { ok: true, models: ['opus-4.8', 'opus-4.7', 'opus-4.6', 'kiro-agent-v1', 'q-developer'] };
  } catch (err: unknown) {
    const errMsg = err instanceof Error ? err.message : String(err);
    return { ok: false, error: `AWS connection failed: ${errMsg}` };
  }
}
export default validateAWS;
