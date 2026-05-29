/* eslint-disable agentverse/no-sideways-capability-imports */
import { useKeyStore } from '@/api/key-store/store';
import { KeyStore } from '@/api/key-store';
import { appFetch } from '@/shell/app-fetch';
import { CreateCanvasIntent } from './types';
import { NLU_SYSTEM_PROMPT, NLU_USER_TEMPLATE } from './nlu-prompt';

export async function extractIntent(transcript: string): Promise<CreateCanvasIntent> {
  const validated = useKeyStore.getState().validated;

  let provider: 'google' | 'openai' | 'anthropic' | null = null;
  let model = '';

  if (validated.includes('google')) {
    provider = 'google';
    const models = useKeyStore.getState().cachedModels['google'] || [];
    model = models.find((m) => m.includes('flash')) || 'gemini-1.5-flash';
  } else if (validated.includes('openai')) {
    provider = 'openai';
    const models = useKeyStore.getState().cachedModels['openai'] || [];
    model = models.find((m) => m.includes('mini')) || 'gpt-4o-mini';
  } else if (validated.includes('anthropic')) {
    provider = 'anthropic';
    const models = useKeyStore.getState().cachedModels['anthropic'] || [];
    model = models.find((m) => m.includes('haiku')) || 'claude-3-5-haiku-20241022';
  }

  if (!provider) {
    throw new Error('No validated LLM provider available');
  }

  const keyRecord = await KeyStore.get(provider);
  const apiKey = keyRecord?.keys['apiKey'] || '';
  if (!apiKey) {
    throw new Error(`API key for ${provider} is missing`);
  }

  const controller = new AbortController();
  const timeoutId = setTimeout(() => {
    controller.abort();
  }, 3000);

  try {
    let url = '';
    let options: RequestInit = {};

    const systemPrompt = NLU_SYSTEM_PROMPT;
    const userPrompt = NLU_USER_TEMPLATE(transcript);

    if (provider === 'google') {
      const cleanModel = model.startsWith('models/') ? model : `models/${model}`;
      url = `https://generativelanguage.googleapis.com/v1beta/${cleanModel}:generateContent?key=${apiKey}`;
      options = {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          contents: [
            {
              parts: [{ text: systemPrompt }, { text: userPrompt }],
            },
          ],
          generationConfig: {
            responseMimeType: 'application/json',
          },
        }),
        signal: controller.signal,
      };
    } else if (provider === 'openai') {
      url = 'https://api.openai.com/v1/chat/completions';
      options = {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${apiKey}`,
        },
        body: JSON.stringify({
          model,
          messages: [
            { role: 'system', content: systemPrompt },
            { role: 'user', content: userPrompt },
          ],
          response_format: { type: 'json_object' },
        }),
        signal: controller.signal,
      };
    } else {
      url = 'https://api.anthropic.com/v1/messages';
      options = {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': apiKey,
          'anthropic-version': '2023-06-01',
        },
        body: JSON.stringify({
          model,
          max_tokens: 1000,
          system: systemPrompt,
          messages: [{ role: 'user', content: userPrompt }],
          tools: [
            {
              name: 'create_canvas',
              description: 'Create a canvas of agents and handoff edges',
              input_schema: {
                type: 'object',
                properties: {
                  name: { type: 'string' },
                  nodes: {
                    type: 'array',
                    items: {
                      type: 'object',
                      properties: {
                        display_name: { type: 'string' },
                        role: { type: 'string', enum: ['supervisor', 'developer', 'reviewer', 'custom'] },
                        provider: { type: 'string' },
                      },
                      required: ['display_name', 'role', 'provider'],
                    },
                  },
                  edges: {
                    type: 'array',
                    items: {
                      type: 'object',
                      properties: {
                        from: { type: 'string' },
                        to: { type: 'string' },
                        type: { type: 'string', enum: ['handoff', 'assign', 'send_message'] },
                      },
                      required: ['from', 'to', 'type'],
                    },
                  },
                  confidence: { type: 'number' },
                },
                required: ['name', 'nodes', 'edges'],
              },
            },
          ],
          tool_choice: { type: 'tool', name: 'create_canvas' },
        }),
        signal: controller.signal,
      };
    }

    const res = await appFetch(url, options);
    clearTimeout(timeoutId);

    if (!res.ok) {
      const errText = await res.text();
      throw new Error(`LLM extraction failed (${res.status}): ${errText}`);
    }

    const json = await res.json();
    let textResult = '';

    if (provider === 'google') {
      textResult = json.candidates?.[0]?.content?.parts?.[0]?.text || '';
    } else if (provider === 'openai') {
      textResult = json.choices?.[0]?.message?.content || '';
    } else {
      const toolUseBlock = (json.content as Array<{ type: string; name: string; input?: unknown }> | undefined)?.find(
        (block) => block.type === 'tool_use' && block.name === 'create_canvas'
      );
      if (toolUseBlock) {
        return toolUseBlock.input as CreateCanvasIntent;
      }
      throw new Error('Anthropic did not invoke the create_canvas tool');
    }

    let cleanedText = textResult.trim();
    if (cleanedText.startsWith('```')) {
      cleanedText = cleanedText.replace(/^```(json)?/, '').replace(/```$/, '').trim();
    }

    const intent = JSON.parse(cleanedText) as CreateCanvasIntent;
    return intent;
  } catch (err) {
    clearTimeout(timeoutId);
    if (err && typeof err === 'object' && 'name' in err && err.name === 'AbortError') {
      throw new Error('NLU_TIMEOUT');
    }
    throw err;
  }
}
